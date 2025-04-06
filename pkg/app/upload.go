package app

import "net/http"
import "log/slog"
import "io"
import "image"
import "image/color"
import "image/jpeg"
import _ "image/png"
import _ "image/gif"

// import "mime/multipart"
import "bytes"

import "golang.org/x/image/draw"

func resizeImageJPEG(input io.Reader, width, height, quality int) (*bytes.Buffer, error) {
	output := &bytes.Buffer{}

	src, _, err := image.Decode(input)

	if err != nil {
		return nil, err
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill image with transparent white first
	tw := color.RGBA{255, 255, 255, 0}
	draw.Draw(dst, dst.Bounds(), &image.Uniform{tw}, image.ZP, draw.Src)
	// Then draw image
	draw.NearestNeighbor.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	err = jpeg.Encode(output, dst, &jpeg.Options{Quality: quality})
	return output, err
}

const maxImageSize = 4000 * 1000 * 1000

func (q *Qrochet) getUpload(wr http.ResponseWriter, req *http.Request) {
	var err error
	v := q.view()

	if !v.IsLoggedIn(wr, req) {
		// XXX might have to do some permission checking later to not display
		// paid images
	}

	id := req.PathValue("id")

	image, err := q.Repository.Image.Get(req.Context(), id)
	if err != nil {
		slog.Error("getUpload", "err", err)
		wr.WriteHeader(http.StatusNotFound)
		return
	}

	// XXX Also support other uploads.
	wr.Header().Set("Content-Type", "image/jpeg")
	// XXX: Should provide the size. wr.Header().Set("Content-Length", image.Size)
	io.Copy(wr, image.ReadCloser)
	image.ReadCloser.Close()
	return
}
