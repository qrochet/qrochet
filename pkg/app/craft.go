package app

import "net/http"
import "strconv"
import "log/slog"
import "io"
import "mime/multipart"

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
	All         chan model.Craft
}

func (q *Qrochet) getMyCraft(wr http.ResponseWriter, req *http.Request) {
	var err error
	v := q.view()
	if !v.IsLoggedIn(wr, req) {
		v.DisplayError(wr, req, "Please log in.")
		return
	}

	v.Craft.All, err = q.Repository.Craft.AllForUserID(req.Context(), v.Session.UserID)
	if !v.IsLoggedIn(wr, req) {
		slog.Error("getMyCraft", "err", err)
		v.DisplayError(wr, req, "No crafts.")
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
		craft.UserID = v.Session.UserID

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
