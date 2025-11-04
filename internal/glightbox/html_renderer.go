package glightbox

import (
	"bytes"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"github.com/SayaAndy/saya-today-web/internal/tailwind"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

type GLightboxHTMLRenderer struct {
	html.Config
	md            goldmark.Markdown
	anchorMatchRe *regexp.Regexp
}

func NewGLightboxHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &GLightboxHTMLRenderer{
		Config: html.NewConfig(),
		md: goldmark.New(
			goldmark.WithParserOptions(
				parser.WithAutoHeadingID(),
				parser.WithAttribute(),
			),
			goldmark.WithRenderer(
				renderer.NewRenderer(
					renderer.WithNodeRenderers(
						util.Prioritized(tailwind.NewCustomLinkRenderer(
							html.WithUnsafe(), html.WithHardWraps(), html.WithXHTML(),
						), 50),
						util.Prioritized(html.NewRenderer(
							html.WithUnsafe(), html.WithHardWraps(), html.WithXHTML(),
						), 100),
					),
				),
			),
		),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	r.anchorMatchRe = regexp.MustCompile(`(?s)<\s*a(\s+[^<]*)(href\s*=\s*["'].*?["'])\s*([^<]*)>(.*?)<\s*\/\s*a\s*>`)
	return r
}

func (r *GLightboxHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindGLightboxBlock, r.renderGLightbox)
}

func (r *GLightboxHTMLRenderer) renderGLightbox(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		gallery := n.(*GLightboxBlock)

		if len(gallery.Images) == 0 {
			return ast.WalkContinue, nil
		}

		galleryID := generateDivId(8)

		var elements []string
		for i, img := range gallery.Images {
			imageUrlSegments := strings.Split(img.URL, ".")
			imageUrlWithoutExt := strings.Join(imageUrlSegments[:len(imageUrlSegments)-1], ".")
			imageUrlParts := strings.Split(imageUrlWithoutExt, "/")
			imageNameParts := strings.Split(imageUrlParts[len(imageUrlParts)-1], "-")

			var captionBuf bytes.Buffer
			captionHTML := img.Caption
			if err := r.md.Convert(img.Caption, &captionBuf); err == nil {
				captionHTML = captionBuf.Bytes()
				captionHTML = bytes.TrimPrefix(captionHTML, []byte("<p>"))
				captionHTML = bytes.TrimSuffix(captionHTML, []byte("</p>\n"))
				captionHTML = bytes.TrimSuffix(captionHTML, []byte("</p>"))
			}

			dayDate, _ := time.Parse("20060102 150405", imageNameParts[len(imageNameParts)-2]+" "+imageNameParts[len(imageNameParts)-1])
			dayDate = dayDate.In(gallery.Location)

			glightboxDescId := ""
			if len(captionHTML) != 0 {
				glightboxDescId = fmt.Sprintf("glightbox-desc-%s-%d", galleryID, i)
			}

			dataDescriptionAttribute := ""
			if glightboxDescId != "" {
				dataDescriptionAttribute = fmt.Sprintf("data-description=\".%s\"", glightboxDescId)
			}

			tagClassList := make([]string, 0, len(img.Tags))
			for _, tag := range img.Tags {
				switch tag {
				case "2x":
					tagClassList = append(tagClassList, "grid-item-2x")
				}
			}

			if len(img.Caption) > 0 {
				tagClassList = append(tagClassList, "grid-tooltip")
			}

			anchorlessCaptionHTML := r.anchorMatchRe.ReplaceAll(captionHTML, []byte("<span class=\"linklike\" $1 $3>$4</span>"))

			elements = append(elements, fmt.Sprintf(`
	<a href="https://f003.backblazeb2.com/file/sayana-photos/full/%s" class="glightbox grid-item %s grid-item-%s p-1"
	    data-gallery="gallery" data-title="%s" %s>
		<picture>
			<source media="(width < 800px)" srcset="https://f003.backblazeb2.com/file/sayana-photos/webp-320p/%s.webp" />
			<source media="(width < 2400px)" srcset="https://f003.backblazeb2.com/file/sayana-photos/webp-560p/%s.webp" />
			<source media="(width < 3200px)" srcset="https://f003.backblazeb2.com/file/sayana-photos/webp-800p/%s.webp" />
			<source media="(width < 4000px)" srcset="https://f003.backblazeb2.com/file/sayana-photos/webp-1200p/%s.webp" />
			<source media="(width >= 4000px)" srcset="https://f003.backblazeb2.com/file/sayana-photos/webp-1600p/%s.webp" />
			<img src="https://f003.backblazeb2.com/file/sayana-photos/webp-560p/%s.webp" />
		</picture>
		<span class="grid-tooltip-text"><p>%s</p></span>
		<span class="grid-item-index">%d</span>
	</a>`, img.URL, strings.Join(tagClassList, " "), galleryID, dayDate.Format("2006-01-02 15:04:05 -07:00"), dataDescriptionAttribute, imageUrlWithoutExt, imageUrlWithoutExt, imageUrlWithoutExt, imageUrlWithoutExt, imageUrlWithoutExt, imageUrlWithoutExt, anchorlessCaptionHTML, i+1))

			if glightboxDescId != "" {
				elements = append(elements, fmt.Sprintf(`
	<div class="glightbox-desc rounded-4 %s">
		<p>%s</p>
	</div>`, glightboxDescId, captionHTML))
			}
		}

		w.WriteString(fmt.Sprintf(`
<div class="items-center flex flex-col">
	<hr class="border-t-3 border-dotted border-main-hard mt-1 mb-2 w-[80%%] ml-auto mr-auto">
	<div class="grid masonry-grid-%s">
		<div class="grid-sizer grid-sizer-%s"></div>
		%s
	</div>
	<hr class="border-t-3 border-dotted border-main-hard mt-1 mb-2 w-[80%%] ml-auto mr-auto">
</div>`, galleryID, galleryID, strings.Join(elements, "\n")))

		w.WriteString(fmt.Sprintf(`
<script>
	var pckry_%s = new Packery('.masonry-grid-%s', {
		itemSelector: '.grid-item-%s',
		columnWidth: '.grid-sizer-%s',
		percentPosition: false
	});

	var imgLoad_%s_timer;
	var imgLoad_%s = imagesLoaded('.masonry-grid-%s');

	function initMasonryLayout_%s(tm) {
		clearTimeout(imgLoad_%s_timer);
		imgLoad_%s_timer = setTimeout(() => pckry_%s.layout(), 100);
	}

	imgLoad_%s.on('progress', initMasonryLayout_%s);

	window.addEventListener('resize', initMasonryLayout_%s);

	document.addEventListener('popout', (e) => {
		window.removeEventListener('resize', initMasonryLayout_%s);
		imgLoad_%s.off('progress', initMasonryLayout_%s);
		initMasonryLayout_%s = null;
		imgLoad_%s = null;
	}, {once: true});
</script>`, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID))
	}

	return ast.WalkContinue, nil
}

func generateDivId(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano())) // Seed with current time
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
