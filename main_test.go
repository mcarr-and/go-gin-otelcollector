package main

import (
	"encoding/json"
	"errors"
	"example/go-gin-example/models"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	//Set Gin to Test Mode
	gin.SetMode(gin.TestMode)

	// Run the other tests
	os.Exit(m.Run())
}

func Test_getAllAlbums(t *testing.T) {

	router := setupRouter()
	w := httptest.NewRecorder()
	var albums []models.Album
	req, _ := http.NewRequest(http.MethodGet, "/albums", nil)

	router.ServeHTTP(w, req)

	if err := json.Unmarshal(w.Body.Bytes(), &albums); err != nil {
		assert.Fail(t, "json unmarshal fail", "should be []Albums ", albums)
	}

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, listAlbums(), albums)
}

func Test_getAlbumById(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	var album models.Album

	req, _ := http.NewRequest(http.MethodGet, "/albums/2", nil)
	router.ServeHTTP(w, req)

	body := w.Body.Bytes()
	if err := json.Unmarshal(body, &album); err != nil {
		assert.Fail(t, "json unmarshal fail", "Should be Album ", string(body))
	}

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, listAlbums()[1], album)
	assert.Equal(t, listAlbums()[1].Title, album.Title)
}

func Test_getAlbumById_BadId(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	var serverError models.ServerError

	req, _ := http.NewRequest(http.MethodGet, "/albums/X", nil)
	router.ServeHTTP(w, req)

	body := w.Body.Bytes()
	if err := json.Unmarshal(body, &serverError); err != nil {
		assert.Fail(t, "json unmarshal fail", "Should be Album ", string(body))
	}

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "Album ID [X] is not a valid number", serverError.Message)
}

func Test_getAlbumById_NotFound(t *testing.T) {
	router := setupRouter()
	w := httptest.NewRecorder()
	var serverError models.ServerError
	albumID := 5666

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s%v", "/albums/", albumID), nil)
	router.ServeHTTP(w, req)

	body := w.Body.Bytes()
	if err := json.Unmarshal(body, &serverError); err != nil {
		assert.Fail(t, "json unmarshalling fail", "Should be ServerError ", string(body))
	}

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, fmt.Sprintf("%s [%v] %s", "Album", albumID, "not found"), serverError.Message)
}

func Test_postAlbum(t *testing.T) {
	resetAlbums()
	router := setupRouter()
	w := httptest.NewRecorder()
	var album models.Album

	albumBody := `{"id": 10, "title": "The Ozzman Cometh", "artist": "Black Sabbath", "price": 66.60}`
	expectedAlbum := models.Album{ID: 10, Title: "The Ozzman Cometh", Artist: "Black Sabbath", Price: 66.60}
	req, _ := http.NewRequest(http.MethodPost, "/albums", strings.NewReader(albumBody))

	assert.Equal(t, len(listAlbums()), 3)
	router.ServeHTTP(w, req)
	body := w.Body.Bytes()

	if err := json.Unmarshal(body, &album); err != nil {
		assert.Fail(t, "json unmarshalling fail", "Should be an Album ", string(body))
	}

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, album, expectedAlbum)
	assert.Equal(t, len(listAlbums()), 4)
}

func Test_postAlbum_BadRequest_BadJSON_MissingValues(t *testing.T) {
	resetAlbums()
	router := setupRouter()
	w := httptest.NewRecorder()
	album := `{"xid": 10, "titlex": "Blue Train", "artistx": "Lead Belly", "pricex": 56.99, "X": "asdf"}`
	var serverError models.ServerError
	req, _ := http.NewRequest(http.MethodPost, "/albums", strings.NewReader(album))
	router.ServeHTTP(w, req)
	body := w.Body.Bytes()

	if err := json.Unmarshal(body, &serverError); err != nil {
		var ve validator.ValidationErrors
		errors.As(err, &ve)
		assert.Fail(t, "json unmarshalling fail", "should be ServerError ", ve.Error(), string(body))
	}
	assert.Equal(t, 4, len(serverError.BindingErrors))
	assert.Equal(t, "title", serverError.BindingErrors[1].Field)
	assert.Equal(t, "required field", serverError.BindingErrors[1].Message)
	assert.Equal(t, "artist", serverError.BindingErrors[2].Field)
	assert.Equal(t, "required field", serverError.BindingErrors[2].Message)
	assert.Equal(t, "price", serverError.BindingErrors[3].Field)
	assert.Equal(t, "required field", serverError.BindingErrors[3].Message)
	assert.Equal(t, len(listAlbums()), 3)
}

func Test_postAlbum_BadRequest_BadJSON_MinValues(t *testing.T) {
	resetAlbums()
	router := setupRouter()
	w := httptest.NewRecorder()
	album := `{"id": -1, "title": "a", "artist": "z", "price": -0.1}`
	var serverError models.ServerError
	req, _ := http.NewRequest(http.MethodPost, "/albums", strings.NewReader(album))
	router.ServeHTTP(w, req)
	body := w.Body.Bytes()

	if err := json.Unmarshal(body, &serverError); err != nil {
		var ve validator.ValidationErrors
		errors.As(err, &ve)
		assert.Fail(t, "json unmarshalling fail", "should be ServerError ", ve.Error(), string(body))
	}
	assert.Equal(t, 4, len(serverError.BindingErrors))
	assert.Equal(t, "id", serverError.BindingErrors[0].Field)
	assert.Equal(t, "below minimum value", serverError.BindingErrors[0].Message)
	assert.Equal(t, "title", serverError.BindingErrors[1].Field)
	assert.Equal(t, "below minimum value", serverError.BindingErrors[1].Message)
	assert.Equal(t, "artist", serverError.BindingErrors[2].Field)
	assert.Equal(t, "below minimum value", serverError.BindingErrors[2].Message)
	assert.Equal(t, "price", serverError.BindingErrors[3].Field)
	assert.Equal(t, "below minimum value", serverError.BindingErrors[3].Message)
	assert.Equal(t, len(listAlbums()), 3)
}

func Benchmark_getAllAlbums(b *testing.B) {
	router := setupRouter()
	w := httptest.NewRecorder()
	var albums []models.Album
	req, _ := http.NewRequest(http.MethodGet, "/albums", nil)

	for i := 0; i < b.N; i++ {
		router.ServeHTTP(w, req)
		body := w.Body.Bytes()
		if err := json.Unmarshal(body, &albums); err != nil {
			assert.Fail(b, "json unmarshalling fail", "should be []Album ", string(body))
		}
		w.Body.Reset() //get requests need resets else the returned body is concatenated
	}
}

func Benchmark_getAlbumById(b *testing.B) {
	router := setupRouter()
	w := httptest.NewRecorder()
	var album models.Album
	req, _ := http.NewRequest(http.MethodGet, "/albums/2", nil)

	for i := 0; i < b.N; i++ {
		router.ServeHTTP(w, req)
		body := w.Body.Bytes()
		if err := json.Unmarshal(body, &album); err != nil {
			assert.Fail(b, "json unmarshalling fail", "should be Album ", string(body))
		}
		w.Body.Reset() //get requests need resets else the returned body is concatenated
	}
}

func Benchmark_getAlbumById_BadRequest(b *testing.B) {
	router := setupRouter()
	w := httptest.NewRecorder()
	var serverError models.ServerError
	req, _ := http.NewRequest(http.MethodGet, "/albums/5666", nil)

	for i := 0; i < b.N; i++ {
		router.ServeHTTP(w, req)
		body := w.Body.Bytes()
		if err := json.Unmarshal(body, &serverError); err != nil {
			assert.Fail(b, "json unmarshalling fail", "should be ServerError ", string(body))
		}
		w.Body.Reset() //get requests need resets else the returned body is concatenated
	}
}

func Benchmark_postAlbum(b *testing.B) {
	router := setupRouter()
	w := httptest.NewRecorder()
	albumJson := `{"id": "10", "title": "The Ozzman Cometh", "artist": "Black Sabbath", "price": 56.99}`
	req, _ := http.NewRequest(http.MethodPost, "/albums", strings.NewReader(albumJson))
	var albumReturned models.Album

	for i := 0; i < b.N; i++ {
		router.ServeHTTP(w, req)
		bodyReturned := w.Body.Bytes()
		if err := json.Unmarshal(bodyReturned, &albumReturned); err != nil {
			assert.Fail(b, "json unmarshalling fail", "should be Album ", string(bodyReturned))
		}
		w.Body.Reset()
	}
}

func Benchmark_postAlbum_BadRequest_BadJson(b *testing.B) {
	router := setupRouter()
	w := httptest.NewRecorder()
	var returnedError models.ServerError
	albumJson := `{"xid": "10", "titlex": "Blue Train", "artistx": "John Coltrane", "pricex": 56.99, "X": "asdf"}`
	req, _ := http.NewRequest(http.MethodPost, "/albums", strings.NewReader(albumJson))

	for i := 0; i < b.N; i++ {
		router.ServeHTTP(w, req)
		bodyReturned := w.Body.Bytes()
		if err := json.Unmarshal(bodyReturned, &returnedError); err != nil {
			assert.Fail(b, "json unmarshalling fail", "Should be ServerError ", string(bodyReturned))
		}
		w.Body.Reset()
	}
}
