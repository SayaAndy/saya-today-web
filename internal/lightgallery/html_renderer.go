package lightgallery

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

type LightGalleryHTMLRenderer struct {
	html.Config
}

func NewLightGalleryHTMLRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &LightGalleryHTMLRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *LightGalleryHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(KindLightGalleryBlock, r.renderLightGallery)
}

func (r *LightGalleryHTMLRenderer) renderLightGallery(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		gallery := n.(*LightGalleryBlock)

		if len(gallery.Images) == 0 {
			return ast.WalkContinue, nil
		}

		galleryID := generateDivId(4)

		w.WriteString(fmt.Sprintf(`
<div class="justify-content-center display-block m-2">
	<hr class="border-t-4 border-dotted border-main-dark mt-vmin0-8 mb-vmin0-8 w-p80 ml-auto mr-auto">
	<div id="lg-%s" class="inline-gallery-container relative ml-auto mr-auto"></div>
	<hr class="border-t-4 border-dotted border-main-dark mt-vmin0-8 mb-vmin0-8 w-p80 ml-auto mr-auto">
</div>`, galleryID))

		var dynamicElements []string
		for _, img := range gallery.Images {
			imageUrlSegments := strings.Split(img.URL, ".")
			imageUrlWithoutExt := strings.Join(imageUrlSegments[:len(imageUrlSegments)-1], ".")
			imageNameParts := strings.Split(imageUrlWithoutExt, "-")

			dayDate, _ := time.Parse("20060102 150405", imageNameParts[len(imageNameParts)-2]+" "+imageNameParts[len(imageNameParts)-1])
			dynamicElements = append(dynamicElements, fmt.Sprintf(`{
				src:
					"https://f003.backblazeb2.com/file/sayana-photos/full/%s",
				downloadUrl:
					"https://f003.backblazeb2.com/file/sayana-photos/full/%s",
				alt: "%s",
				sources: [{
					srcset: "https://f003.backblazeb2.com/file/sayana-photos/thumbnails/%s.webp",
					media: "(max-width: 800px)"
				}],
				thumb:
					"https://f003.backblazeb2.com/file/sayana-photos/thumbnails/%s.webp",
				subHtml: `+"`"+`<div class="flex flex-row light-gallery-captions">
								<p class="grow text-vmax1 text-left lh-0-9 font-spectral text-main-dark">%s</p>
								<p class="text-vmax1 text-right lh-0-9 font-spectral text-secondary">%s</p>
							</div>`+"`"+`
			}`, img.URL, img.URL, img.Caption, imageUrlWithoutExt, imageUrlWithoutExt, img.Caption, dayDate.Format("2006-01-02 15:04:05 -07:00")))
		}

		w.WriteString(fmt.Sprintf(`
<script>
function createLightLibrary%s() {
		const $lgContainer = document.getElementById('lg-%s');
        const inlineGallery = lightGallery($lgContainer, {
			container: $lgContainer,
			dynamic: true,
			dynamicEl: [%s],
			width: "100%%",
            height: "50vmin",
			hash: false,
			closable: false,
			showMaximizeIcon: true,
			appendSubHtmlTo: ".lg-sub-html",
            isMobile: () => { return false; },
			slideDelay: 0,
			plugins: [lgZoom, lgThumbnail],
			thumbWidth: 160,
			thumbHeight: "10vmin",
			thumbMargin: 4
        });

		setTimeout(() => {
			inlineGallery.openGallery();
		}, 200);
}

document.addEventListener('DOMContentLoaded', createLightLibrary%s);

let resizeTimer%s;
window.addEventListener("resize", () => {
	clearTimeout(resizeTimer%s);
	resizeTimer%s = setTimeout(() => {
		const $lgContainer = document.getElementById('lg-%s');
		$lgContainer.innerHTML = "";
		createLightLibrary%s();
	}, 1000);
});
</script>`, galleryID, galleryID, strings.Join(dynamicElements, ","), galleryID, galleryID, galleryID, galleryID, galleryID, galleryID))
	}

	return ast.WalkContinue, nil
}

func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
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
