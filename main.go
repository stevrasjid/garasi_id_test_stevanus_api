package main

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
)

type DataType struct {
	Id	int
	File multipart.File
	Filename string
	Title string
}

type SucccessResponseType struct {
	Message string `json:"message"`
	Data []string `json:"data"`
}

type ErrorResponseType struct {
	Message string `json:"message"`
}

type AppError struct {
	Err  error
	Code int
}

func main() {
	r := chi.NewRouter()
	r.Post("/api/uploadFile", uploadFileController)
	r.Get("/api/getFileUrl", getFileUrl)
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"OPTIONS", "GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"*"}, 
		MaxAge:           300,
		AllowCredentials: false,
	})
	handler := c.Handler(r)
	http.ListenAndServe(":8080", handler)
}

func getFileUrl(w http.ResponseWriter, r *http.Request) {
	path := "./uploads/"
	files, err := os.ReadDir(path)
	if err != nil {
		errorResponse := SetError("Unable to parse form", http.StatusBadRequest)
		ErrorResponse(w, errorResponse.Code, errorResponse.Err)
		return
	}
	var imageList []string;
	for _, file := range files {
		if !file.IsDir() {
			fileExt := filepath.Ext(file.Name())
			errorResponse := CheckExtension(fileExt, 0)
			if errorResponse != nil {
				break
			}
			imageList = append(imageList, "/uploads/"+file.Name())
		}
	}
	
	SuccessResponse(w, imageList)
}

func uploadFileController(w http.ResponseWriter, r *http.Request) {
	var errorResponse *AppError
	var imageList []string;

	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		errorResponse = SetError("Unable to parse form", http.StatusBadRequest)
		ErrorResponse(w, errorResponse.Code, errorResponse.Err)
		return
	} 
	
	if len(r.MultipartForm.File) == 0 {
		errorResponse = SetError("No File was send", http.StatusBadRequest)
		ErrorResponse(w, errorResponse.Code, errorResponse.Err)
		return
	}

	re := regexp.MustCompile(`data\[(\d+)\]\[(\w+)\]`)
	filesMap := make(map[int]*DataType)

	path := "./uploads/"
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(path, os.ModePerm)
		}
	}

	for key, values := range r.Form {
		match := re.FindStringSubmatch(key)
		if len(match) == 3 {
			index, _ := strconv.Atoi(match[1])
			field := match[2]

			if filesMap[index] == nil {
				filesMap[index] = &DataType{}
			}

			if field == "title" {
				filesMap[index].Title = values[0] 
			}
		}
	}

	if r.MultipartForm != nil {
		for key, fileHeaders := range r.MultipartForm.File {
			match := re.FindStringSubmatch(key)
			if len(match) == 3 {
				index, _ := strconv.Atoi(match[1])
				field := match[2]
				if field == "image" && len(fileHeaders) > 0 {
					fileHeader := fileHeaders[0]
					file, err := fileHeader.Open()
					if err != nil {
						errorResponse = SetError("Error opening file", http.StatusInternalServerError)
						break
					}
					defer file.Close()
					fileExt := filepath.Ext(fileHeader.Filename)
					errorResponse = CheckExtension(fileExt, index)
					if errorResponse != nil {
						break
					}
				}	
			}
		} 

		if errorResponse != nil {
			ErrorResponse(w, errorResponse.Code, errorResponse.Err)
			return
		}

		for key, fileHeaders := range r.MultipartForm.File {
			match := re.FindStringSubmatch(key)
			if len(match) == 3 {
				index, _ := strconv.Atoi(match[1])
				fileHeader := fileHeaders[0]
				file, _ := fileHeader.Open()
				defer file.Close()

				fileName := filesMap[index].Title + "_" + time.Now().Format("20060102150405") + fileHeader.Filename 
				dst, err := os.Create(path + fileName)
				if err != nil {
					errorResponse = SetError("Error saving file", http.StatusInternalServerError)
					break
				}
				defer dst.Close()
				
				_, err = io.Copy(dst, file)
				if err != nil {
					errorResponse = SetError("Error copying file", http.StatusInternalServerError)
					break
				}
				imageList = append(imageList, "/uploads/"+fileName)
			}
		}
	}
	
	if errorResponse != nil {
		ErrorResponse(w, errorResponse.Code, errorResponse.Err)
	} else {
		SuccessResponse(w, imageList)
	}
}

func SetError(messageError string, code int) *AppError {
	return &AppError{Err: fmt.Errorf(messageError), Code: code}
}

func SuccessResponse(w http.ResponseWriter, imageList []string) {
	w.Header().Add("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)

	json.NewEncoder(w).Encode(SucccessResponseType{
		Message: "success",
		Data: imageList,
	})
}

func ErrorResponse(w http.ResponseWriter, status int, err error) {
	w.Header().Add("Content-type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(ErrorResponseType{
		Message: "error : " + err.Error(),
	})
}

func CheckExtension(ext string, index int) *AppError {
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif":
		return nil
	default:
		errorResponse := SetError(fmt.Sprintf("File extension only accept png, jpg, jpeg, and gif at row number %d", index+1), http.StatusBadRequest)
		return errorResponse
}
}

