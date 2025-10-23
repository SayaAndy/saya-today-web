package mailer

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SayaAndy/saya-today-web/internal/templatemanager"
	"github.com/SayaAndy/saya-today-web/locale"
	"github.com/dgraph-io/ristretto/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/wneessen/go-mail"
	"golang.org/x/crypto/argon2"
)

type Mailer struct {
	verificationCodes *ristretto.Cache[uint64, string]
	db                *sql.DB
	tm                *templatemanager.TemplateManager
	mailClient        *mail.Client
	clientHost        string
	mailAddress       string
	publicName        string
	salt              []byte

	lostMailMap map[string]struct {
		Dur        time.Duration
		End        time.Time
		CodeExpiry time.Time
	}
	lostMailMapMutex sync.RWMutex

	hashMap      map[string][]byte
	hashMapMutex sync.RWMutex

	l map[string]*locale.LocaleConfig
}

type SubscriptionType int

const (
	All SubscriptionType = iota
	None
	Specific
)

func NewMailer(db *sql.DB, clientHost string, mailHost string, publicName string, mailAddress string, username string, password string, salt []byte, localization map[string]*locale.LocaleConfig) (*Mailer, error) {
	verificationCodes, err := ristretto.NewCache(&ristretto.Config[uint64, string]{
		NumCounters:            10000,
		MaxCost:                1 << 20, // 1 MB
		BufferItems:            64,
		TtlTickerDurationInSec: 3600,
	})
	if err != nil {
		return nil, fmt.Errorf("fail to initialize cache for verification codes: %w", err)
	}

	tm, err := templatemanager.NewTemplateManager(templatemanager.TemplateManagerTemplates{
		Name:  "new-post",
		Files: []string{"views/layouts/general-mail.html", "views/messages/new-post.html"},
	}, templatemanager.TemplateManagerTemplates{
		Name:  "verify-email",
		Files: []string{"views/layouts/general-mail.html", "views/messages/verify-email.html"},
	})
	if err != nil {
		return nil, fmt.Errorf("fail to initialize template manager for message templating: %w", err)
	}

	mailClient, err := mail.NewClient(mailHost,
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover), mail.WithTLSPortPolicy(mail.TLSMandatory),
		mail.WithUsername(username), mail.WithPassword(password),
	)
	if err != nil {
		return nil, fmt.Errorf("fail to initialize mail client: %w", err)
	}

	return &Mailer{
		verificationCodes: verificationCodes,
		db:                db,
		clientHost:        clientHost,
		tm:                tm,
		mailClient:        mailClient,
		mailAddress:       mailAddress,
		publicName:        publicName,
		salt:              salt,
		hashMap:           make(map[string][]byte),
		lostMailMap: make(map[string]struct {
			Dur        time.Duration
			End        time.Time
			CodeExpiry time.Time
		}, 0),
		l: localization}, nil
}

func (m *Mailer) GetHash(id string) []byte {
	m.hashMapMutex.RLock()
	if val, ok := m.hashMap[id]; ok {
		m.hashMapMutex.RUnlock()
		slog.Debug("gave an old hash", slog.String("hash", base64.RawStdEncoding.EncodeToString(val)))
		return val
	}
	m.hashMapMutex.RUnlock()

	m.hashMapMutex.Lock()
	defer m.hashMapMutex.Unlock()

	if val, ok := m.hashMap[id]; ok {
		slog.Debug("gave a newly generated hash", slog.String("hash", base64.RawStdEncoding.EncodeToString(val)))
		return val
	}

	m.hashMap[id] = argon2.IDKey([]byte(id), m.salt, 1, 64*1024, 4, 32)
	slog.Debug("generated hash", slog.String("hash", base64.RawStdEncoding.EncodeToString(m.hashMap[id])))
	return m.hashMap[id]
}

func (m *Mailer) IsAllowedToRetryVerification(userId string) (retryAllowed bool, whenAllowed time.Time, codeExpiry time.Time) {
	m.lostMailMapMutex.RLock()
	defer m.lostMailMapMutex.RUnlock()
	previous, ok := m.lostMailMap[userId]
	if ok && previous.End.After(time.Now()) {
		return false, previous.End, previous.CodeExpiry
	}
	return true, time.Time{}, previous.CodeExpiry
}

func (m *Mailer) MailIsTaken(email string) (bool, error) {
	tx, err := m.db.Begin()
	if err != nil {
		return false, fmt.Errorf("failed to initialize transaction with db: %s", err)
	}

	var rows *sql.Rows
	if rows, err = tx.Query(`SELECT email FROM user_email_table WHERE email=? LIMIT 1;`, email); err != nil {
		tx.Rollback()
		return false, fmt.Errorf("failed to query user-email settings in db: %s", err)
	}
	defer tx.Commit()
	defer rows.Close()

	isTaken := rows.Next()
	return isTaken, nil
}

func (m *Mailer) GetInfo(userId string) (email string, lang string, err error) {
	hash := m.GetHash(userId)

	tx, err := m.db.Begin()
	if err != nil {
		return "", "", fmt.Errorf("failed to initialize transaction with db: %s", err)
	}

	var rows *sql.Rows
	if rows, err = tx.Query(`SELECT email, lang FROM user_email_table WHERE user_id=? LIMIT 1;`, hash); err != nil {
		tx.Rollback()
		return "", "", fmt.Errorf("failed to query user-email settings in db: %s", err)
	}
	defer tx.Commit()
	defer rows.Close()

	if !rows.Next() {
		return "", "", nil
	}

	if err = rows.Scan(&email, &lang); err != nil {
		return "", "", fmt.Errorf("failed to scan the result from user-email settings query: %s", err)
	}
	return
}

func (m *Mailer) SendVerificationCode(userId string, address string, lang string) error {
	message := mail.NewMsg()

	if err := message.EnvelopeFrom(m.mailAddress); err != nil {
		return fmt.Errorf("failed to set ENVELOPE FROM address: %w", err)
	}
	if err := message.FromFormat(m.publicName, m.mailAddress); err != nil {
		return fmt.Errorf("failed to set formatted FROM address: %w", err)
	}
	if err := message.To(address); err != nil {
		return fmt.Errorf("failed to set TO address: %w", err)
	}

	message.SetMessageID()
	message.SetDate()
	message.SetBulk()

	dur := 1 * time.Minute
	m.lostMailMapMutex.Lock()
	if previous, ok := m.lostMailMap[userId]; ok {
		if time.Now().Before(previous.End) {
			m.lostMailMapMutex.Unlock()
			return fmt.Errorf("user is not allowed to send another verification code until %s", previous.End)
		}
		dur = 2 * m.lostMailMap[userId].Dur
	}
	m.lostMailMap[userId] = struct {
		Dur        time.Duration
		End        time.Time
		CodeExpiry time.Time
	}{Dur: dur, End: time.Now().Add(dur), CodeExpiry: time.Now().Add(time.Hour)}
	m.lostMailMapMutex.Unlock()

	verificationCodeBytes := make([]byte, 8)
	rand.Read(verificationCodeBytes)
	verificationCode := binary.LittleEndian.Uint64(verificationCodeBytes)

	verificationInfo := fmt.Sprintf("%s.%s", base64.RawStdEncoding.EncodeToString([]byte(userId)), base64.RawStdEncoding.EncodeToString([]byte(address)))
	m.verificationCodes.Set(verificationCode, verificationInfo, int64(len(verificationInfo)+8))

	message.Subject(m.l[lang].Mail.VerifyEmail.Subject)

	msg, err := m.tm.Render("verify-email", fiber.Map{
		"L":                m.l[lang],
		"Lang":             lang,
		"VerificationCode": fmt.Sprintf("%X", verificationCode),
		"ClientHost":       m.clientHost,
	})
	if err != nil {
		return fmt.Errorf("failed to render message body: %w", err)
	}

	message.SetBodyString(mail.TypeTextHTML, string(msg))
	if err := m.mailClient.DialAndSend(message); err != nil {
		return fmt.Errorf("failed to send verification code message: %w", err)
	}
	slog.Debug("verification code message successfully delivered", slog.String("address", address), slog.String("user_id", userId))
	return nil
}

func (m *Mailer) Verify(verificationCodeEncoded string, lang string) error {
	verificationCode, err := strconv.ParseUint(verificationCodeEncoded, 16, 64)
	if err != nil {
		return fmt.Errorf("failed to decode verification code from 8-byte hex: %w", err)
	}

	verificationInfo, _ := m.verificationCodes.Get(verificationCode)
	if verificationInfo == "" {
		return fmt.Errorf("failed to get verification info by its code (might be absent, might be empty)")
	}

	verificationSegments := strings.Split(verificationInfo, ".")
	if len(verificationSegments) != 2 {
		m.verificationCodes.Del(verificationCode)
		return fmt.Errorf("invalid format of verification info: expected %d segments, got %d", 2, len(verificationSegments))
	}

	userId, err := base64.RawStdEncoding.DecodeString(verificationSegments[0])
	if err != nil {
		m.verificationCodes.Del(verificationCode)
		delete(m.lostMailMap, verificationSegments[0])
		return fmt.Errorf("could not decode user id: %s", err)
	}

	address, err := base64.RawStdEncoding.DecodeString(verificationSegments[1])
	if err != nil {
		m.verificationCodes.Del(verificationCode)
		delete(m.lostMailMap, verificationSegments[0])
		return fmt.Errorf("could not decode address: %s", err)
	}

	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to initialize transaction with db: %s", err)
	}

	if _, err = tx.Exec(`INSERT INTO user_email_table(user_id, email, lang) VALUES(?, ?, ?)
  ON CONFLICT(user_id) DO UPDATE SET
  	email=excluded.email,
	lang=excluded.lang;`, m.GetHash(string(userId)), address, lang); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to configure user-email settings in db: %s", err)
	}

	if err = tx.Commit(); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to commit transaction to db: %s", err)
	}

	m.verificationCodes.Del(verificationCode)
	delete(m.lostMailMap, verificationSegments[0])
	return nil
}

func (m *Mailer) GetSubscriptions(userId string) (subscriptionType SubscriptionType, tags []string, err error) {
	tx, err := m.db.Begin()
	if err != nil {
		return None, nil, fmt.Errorf("failed to initialize transaction with db: %s", err)
	}

	hash := m.GetHash(userId)

	var rows *sql.Rows
	if rows, err = tx.Query(`SELECT tags FROM subscription_user_to_tags_table WHERE user_id=? LIMIT 1;`, hash); err != nil {
		tx.Rollback()
		return None, nil, fmt.Errorf("failed to query user-to-tags table in db for the user: %s", err)
	}
	defer tx.Commit()
	defer rows.Close()

	if !rows.Next() {
		return None, nil, nil
	}

	tagsString := ""
	if err = rows.Scan(&tagsString); err != nil {
		return None, nil, fmt.Errorf("failed to scan the result from user-to-tags query: %s", err)
	}

	switch tagsString {
	case "":
		return None, nil, nil
	case "_all":
		return All, nil, nil
	default:
		return Specific, strings.Split(tagsString, ","), nil
	}
}

func (m *Mailer) Subscribe(userId string, subscriptionType SubscriptionType, tags ...string) error {
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to initialize transaction with db: %s", err)
	}

	tagsOutput := ""
	switch subscriptionType {
	case All:
		tagsOutput = "_all"
	case None:
		tagsOutput = ""
	case Specific:
		tagsOutput = strings.Join(tags, ",")
	}

	hash := m.GetHash(userId)

	if _, err = tx.Exec(`INSERT INTO subscription_user_to_tags_table(user_id, tags) VALUES(?, ?)
  ON CONFLICT(user_id) DO UPDATE SET
  	tags=excluded.tags;`, hash, tagsOutput); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to configure user-to-tags table in db for the user: %s", err)
	}

	if err = tx.Commit(); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to commit transaction to db: %s", err)
	}

	return nil
}

func SendNewPost() {

}
