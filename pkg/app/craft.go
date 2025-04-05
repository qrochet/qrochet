package app

import "net/http"
import "strconv"
import "log/slog"
import "io"
import "image"
import "image/jpeg"
import _ "image/png"
import _ "image/gif"
import "mime/multipart"
import "bytes"

import "golang.org/x/image/draw"
import "github.com/oklog/ulid/v2"
import "github.com/qrochet/qrochet/pkg/model"

type craft struct {
	ID          string
	Name        string
	Description string
	Submit      bool
	Image       multipart.File
	Header      *multipart.FileHeader
	OK          bool
}

func resizeImageJPEG(input io.Reader, width, height, quality int) (*bytes.Buffer, error) {
	output := &bytes.Buffer{}

	src, _, err := image.Decode(input)

	if err != nil {
		return nil, err
	}

	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.NearestNeighbor.Scale(dst, dst.Rect, src, src.Bounds(), draw.Over, nil)

	jpeg.Encode(output, dst, &jpeg.Options{Quality: quality})
	return output, err
}

const maxImageSize = 4000 * 1000 * 1000

func (q *Qrochet) getMyCraft(wr http.ResponseWriter, req *http.Request) {
	v := q.view()
	if !v.IsLoggedIn(wr, req) {
		v.DisplayError(wr, req, "Please log in.")
		return
	}

	v.Display(wr, req)
	return
}

func (q *Qrochet) postMyCraft(wr http.ResponseWriter, req *http.Request) {
	var err error

	v := q.view()
	if !v.IsLoggedIn(wr, req) {
		v.DisplayError(wr, req, "Please log in.")
		return
	}

	slog.Info("register")
	if req.Method == "POST" {
		err = req.ParseMultipartForm(mpfMaxMemory)
		if err != nil {
			slog.Error("createCraft req.ParseForm", "err", err)
			v.DisplayError(wr, req, "Form error.")
			return
		}
	} else {
		v.DisplayError(wr, req, "Form method must be POST.")
		return
	}

	v.Craft.Name = req.FormValue("name")
	v.Craft.Description = req.FormValue("description")
	v.Craft.Submit, _ = strconv.ParseBool(req.FormValue("submit"))
	v.Craft.Image, v.Craft.Header, err = req.FormFile("image")
	if err != nil {
		slog.Error("createCraft req.FormFile", "err", err)
		v.DisplayError(wr, req, "File upload failed")
		return
	}

	if v.Craft.Header.Size > maxImageSize {
		slog.Error("createCraft Size > maxImageSize", "err", err)
		v.DisplayError(wr, req, "Image too large, maximum 4 MiB.")
		return
	}

	if v.Craft.Submit {
		resized, err := resizeImageJPEG(v.Craft.Image, 640, 640, 90)
		if err != nil {
			slog.Error("resizeImageJPEG", "err", err)
			v.DisplayError(wr, req, "File resizing failed")
			return
		}

		upload := &model.Upload{
			ID:         model.Reference(ulid.Make().String() + ".jpeg"),
			UserID:     v.Session.UserID,
			ReadCloser: io.NopCloser(resized),
		}

		ctx := req.Context()
		upload, err = q.Repository.Image.Put(ctx, upload)
		if err != nil {
			slog.Error("resizeImageJPEG", "err", err)
			v.DisplayError(wr, req, "File save failed")
			return
		}

		craft := &model.Craft{}
		craft.ID = ulid.Make().String()
		craft.Title = v.Craft.Name
		craft.Detail = v.Craft.Description
		craft.Image = upload.ID

		created, err := v.app.Repository.Craft.Put(ctx, craft.ID, *craft)
		if err != nil {
			slog.Error("Craft.Put", "err", err)
			v.DisplayError(wr, req, "Failed to create craft.")
			return
		}
		v.Message("Craft created OK: %s %s", created.Title, created.ID)
		v.Craft.OK = true
		v.Display(wr, req)
		return
	} else {
		slog.Error("Not submitted?", "form", req.Form, "post", req.PostForm)
		v.Display(wr, req)
		return
	}
}
