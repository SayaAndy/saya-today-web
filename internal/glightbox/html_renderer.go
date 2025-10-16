package glightbox

import (
	"bytes"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

type GLightboxHTMLRenderer struct {
	html.Config
	md goldmark.Markdown
}

func NewGLightboxHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &GLightboxHTMLRenderer{
		Config: html.NewConfig(),
		md: goldmark.New(
			goldmark.WithRendererOptions(
				html.WithHardWraps(),
				html.WithXHTML(),
			),
		),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
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

			elements = append(elements, fmt.Sprintf(`
	<a href="https://f003.backblazeb2.com/file/sayana-photos/full/%s" class="glightbox-%s grid-item grid-item-%s"
	    data-gallery="gallery-%s" data-title="%s" %s>
	    <img src="https://f003.backblazeb2.com/file/sayana-photos/webp-320p/%s.webp" class="rounded-md p-0.5" alt="webp thumbnail" />
	</a>`, img.URL, galleryID, galleryID, galleryID, dayDate.Format("2006-01-02 15:04:05 -07:00"), dataDescriptionAttribute, imageUrlWithoutExt))

			if glightboxDescId != "" {
				elements = append(elements, fmt.Sprintf(`
	<div class="glightbox-desc rounded-4 %s">
		<p>%s</p>
	</div>`, glightboxDescId, captionHTML))
			}
		}

		w.WriteString(fmt.Sprintf(`
<div class="justify-content-center display-block m-1">
	<hr class="border-t-3 border-dotted border-main-dark mt-1 mb-2 w-[80%%] ml-auto mr-auto">
	<div class="grid masonry-grid-%s">
		<div class="grid-sizer grid-sizer-%s"></div>
		%s
	</div>
	<hr class="border-t-3 border-dotted border-main-dark mt-1 mb-2 w-[80%%] ml-auto mr-auto">
</div>`, galleryID, galleryID, strings.Join(elements, "\n")))

		w.WriteString(fmt.Sprintf(`
<script>
	var lightbox_%s = GLightbox({
		selector: '.glightbox-%s'
	});

	var msnry_%s = new Masonry('.masonry-grid-%s', {
		itemSelector: '.grid-item-%s',
		columnWidth: '.grid-sizer-%s',
		percentPosition: true
	});

	imagesLoaded('.masonry-grid-%s', function() {
		msnry_%s.layout();
	});
</script>`, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID, galleryID))
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
