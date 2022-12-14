package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// inspired by this for setting up gin & otel to test spans
// https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/instrumentation/github.com/gin-gonic/gin/otelgin/test/gintrace_test.go
//inspired by https://www.thegreatcodeadventure.com/mocking-http-requests-in-golang/

func TestMain(m *testing.M) {
	//Set Gin to Test Mode
	gin.SetMode(gin.TestMode)

	// Run the other tests
	os.Exit(m.Run())
}

// MockClient is the mock client
type MockClient struct {
	Timeout time.Duration
	DoFunc  func(req *http.Request) (*http.Response, error)
}

var (
	// MockResponseFunc fetches the mock client's `Do` func
	MockResponseFunc func(req *http.Request) (*http.Response, error)
)

// Do is the mock client's `Do` func
func (m *MockClient) Do(req *http.Request) (*http.Response, error) {
	return MockResponseFunc(req)
}

func setupTestRouter() (*httptest.ResponseRecorder, *tracetest.SpanRecorder, *gin.Engine) {
	spanRecorder := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder)))
	router := setupRouter()
	testRecorder := httptest.NewRecorder()
	router.Use(otelgin.Middleware("test-otel"))
	return testRecorder, spanRecorder, router
}

func makeKeyMap(attributes []attribute.KeyValue) map[attribute.Key]attribute.Value {
	var attributeMap = make(map[attribute.Key]attribute.Value)
	for _, keyValue := range attributes {
		attributeMap[keyValue.Key] = keyValue.Value
	}
	return attributeMap
}

func Test_getAllAlbums_Success(t *testing.T) {
	testRecorder, spanRecorder, router := setupTestRouter()
	DefaultClient = &MockClient{}

	responseBody := `[{"artist":"Black Sabbath","id":10,"price":66.6,"title":"The Ozzman Cometh"}]`
	body := io.NopCloser(bytes.NewReader([]byte(responseBody)))

	//inject a success message from the server and return a json blob that represents an album
	MockResponseFunc = func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       body,
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "/albums", nil)
	router.ServeHTTP(testRecorder, req)
	b, _ := io.ReadAll(testRecorder.Body)
	returnedBody := string(b)

	assert.Equal(t, http.StatusOK, testRecorder.Code)

	finishedSpans := spanRecorder.Ended()
	assert.Len(t, finishedSpans, 1)

	assert.Equal(t, codes.Ok, finishedSpans[0].Status().Code)
	assert.Equal(t, "", finishedSpans[0].Status().Description)

	assert.Equal(t, 0, len(finishedSpans[0].Events()))

	attributeMap := makeKeyMap(finishedSpans[0].Attributes())
	assert.Equal(t, "200", attributeMap["proxy-service.status_code"].Emit())

	assert.Equal(t, responseBody, returnedBody)
}

func Test_getAllAlbums_Failure_Album_Returns_Error(t *testing.T) {
	testRecorder, spanRecorder, router := setupTestRouter()
	DefaultClient = &MockClient{}

	//inject in failure message to respond with that we could not get to the album-store
	MockResponseFunc = func(*http.Request) (*http.Response, error) {
		return nil, errors.New(
			"Error from web server",
		)
	}

	req, _ := http.NewRequest(http.MethodGet, "/albums", nil)
	router.ServeHTTP(testRecorder, req)
	b, _ := io.ReadAll(testRecorder.Body)
	returnedBody := string(b)

	assert.Equal(t, http.StatusBadRequest, testRecorder.Code)

	finishedSpans := spanRecorder.Ended()
	assert.Len(t, finishedSpans, 1)

	assert.Equal(t, codes.Error, finishedSpans[0].Status().Code)
	assert.Equal(t, "error contacting album-store getAlbums Error from web server", finishedSpans[0].Status().Description)

	assert.Equal(t, 1, len(finishedSpans[0].Events()))
	assert.Equal(t, "error contacting album-store getAlbums Error from web server", finishedSpans[0].Events()[0].Name)

	attributeMap := makeKeyMap(finishedSpans[0].Attributes())
	assert.Equal(t, "400", attributeMap["proxy-service.status_code"].Emit())

	assert.Equal(t, `{"message":"error contacting album-store getAlbums Error from web server"}`, returnedBody)
}

func Test_getAllAlbums_Failure_Malformed_Response(t *testing.T) {
	testRecorder, spanRecorder, router := setupTestRouter()
	DefaultClient = &MockClient{}

	//inject in failure message to respond with that we could not get to the album-store
	responseBody := `[{"artist":"Black Sabbath","id":10,"price":66.6`
	body := io.NopCloser(bytes.NewReader([]byte(responseBody)))

	//inject a failure message from the server and return a json blob that represents an album
	MockResponseFunc = func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       body,
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "/albums", nil)
	router.ServeHTTP(testRecorder, req)
	b, _ := io.ReadAll(testRecorder.Body)
	returnedBody := string(b)

	assert.Equal(t, http.StatusBadRequest, testRecorder.Code)

	finishedSpans := spanRecorder.Ended()
	assert.Len(t, finishedSpans, 1)

	assert.Equal(t, codes.Error, finishedSpans[0].Status().Code)
	assert.Equal(t, "error from album-store getAlbums malformed JSON", finishedSpans[0].Status().Description)

	assert.Equal(t, 1, len(finishedSpans[0].Events()))
	assert.Equal(t, "error from album-store getAlbums malformed JSON", finishedSpans[0].Events()[0].Name)

	attributeMap := makeKeyMap(finishedSpans[0].Attributes())
	assert.Equal(t, "400", attributeMap["proxy-service.status_code"].Emit())
	assert.Equal(t, responseBody, attributeMap["proxy-service.response.body"].Emit())

	assert.Equal(t, `{"message":"error from album-store getAlbums malformed JSON"}`, returnedBody)
}

func Test_getAlbumById_Success(t *testing.T) {
	testRecorder, spanRecorder, router := setupTestRouter()
	DefaultClient = &MockClient{}

	responseBody := `{"artist":"Black Sabbath","id":10,"price":66.6,"title":"The Ozzman Cometh"}`
	body := io.NopCloser(bytes.NewReader([]byte(responseBody)))

	//inject a success message from the server and return a json blob that represents an album
	MockResponseFunc = func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       body,
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "/albums/1", nil)
	router.ServeHTTP(testRecorder, req)
	b, _ := io.ReadAll(testRecorder.Body)
	returnedBody := string(b)

	assert.Equal(t, http.StatusOK, testRecorder.Code)

	finishedSpans := spanRecorder.Ended()
	assert.Len(t, finishedSpans, 1)

	assert.Equal(t, codes.Ok, finishedSpans[0].Status().Code)
	assert.Equal(t, "", finishedSpans[0].Status().Description)

	assert.Equal(t, 0, len(finishedSpans[0].Events()))

	attributeMap := makeKeyMap(finishedSpans[0].Attributes())
	assert.Equal(t, "200", attributeMap["proxy-service.status_code"].Emit())
	assert.Equal(t, "ID=1", attributeMap["proxy-service.request.parameters"].Emit())
	assert.Equal(t, responseBody, attributeMap["proxy-service.response.body"].Emit())
	assert.Equal(t, responseBody, returnedBody)
}

func Test_getAlbumById_Failure_Album_Returns_Error(t *testing.T) {
	testRecorder, spanRecorder, router := setupTestRouter()
	DefaultClient = &MockClient{}

	//inject in failure message to respond with that we could not get to the album-store
	MockResponseFunc = func(*http.Request) (*http.Response, error) {
		return nil, errors.New(
			"Error from web server",
		)
	}

	req, _ := http.NewRequest(http.MethodGet, "/albums/1", nil)
	router.ServeHTTP(testRecorder, req)
	b, _ := io.ReadAll(testRecorder.Body)
	returnedBody := string(b)

	assert.Equal(t, http.StatusBadRequest, testRecorder.Code)

	finishedSpans := spanRecorder.Ended()
	assert.Len(t, finishedSpans, 1)

	assert.Equal(t, codes.Error, finishedSpans[0].Status().Code)
	assert.Equal(t, "error contacting album-store getAlbumById Error from web server", finishedSpans[0].Status().Description)

	assert.Equal(t, 1, len(finishedSpans[0].Events()))
	assert.Equal(t, "error contacting album-store getAlbumById Error from web server", finishedSpans[0].Events()[0].Name)

	attributeMap := makeKeyMap(finishedSpans[0].Attributes())
	assert.Equal(t, "400", attributeMap["proxy-service.status_code"].Emit())
	assert.Equal(t, "ID=1", attributeMap["proxy-service.request.parameters"].Emit())

	assert.Equal(t, `{"message":"error contacting album-store getAlbumById Error from web server"}`, returnedBody)
}

func Test_getAlbumById_Failure_Album_BadId(t *testing.T) {
	testRecorder, spanRecorder, router := setupTestRouter()
	DefaultClient = &MockClient{}

	//inject in failure message to respond with that we could not get to the album-store
	MockResponseFunc = func(*http.Request) (*http.Response, error) {
		return nil, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "/albums/X", nil)
	router.ServeHTTP(testRecorder, req)
	b, _ := io.ReadAll(testRecorder.Body)
	returnedBody := string(b)

	assert.Equal(t, http.StatusBadRequest, testRecorder.Code)

	finishedSpans := spanRecorder.Ended()
	assert.Len(t, finishedSpans, 1)

	assert.Equal(t, codes.Error, finishedSpans[0].Status().Code)
	assert.Equal(t, "error invalid ID [X] requested", finishedSpans[0].Status().Description)

	assert.Equal(t, 1, len(finishedSpans[0].Events()))
	assert.Equal(t, "error invalid ID [X] requested", finishedSpans[0].Events()[0].Name)

	attributeMap := makeKeyMap(finishedSpans[0].Attributes())
	assert.Equal(t, "400", attributeMap["proxy-service.status_code"].Emit())
	assert.Equal(t, "ID=X", attributeMap["proxy-service.request.parameters"].Emit())
	assert.Equal(t, `{"message":"error invalid ID [X] requested"}`, returnedBody)
}

func Test_getAlbumById_Failure_Malformed_Response(t *testing.T) {
	testRecorder, spanRecorder, router := setupTestRouter()
	DefaultClient = &MockClient{}

	//inject in failure message to respond with that we could not get to the album-store
	responseBody := `{"artist":"Black Sabbath","id":10,"price":66.6`
	body := io.NopCloser(bytes.NewReader([]byte(responseBody)))

	//inject a failure message from the server and return a json blob that represents an album
	MockResponseFunc = func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       body,
		}, nil
	}

	req, _ := http.NewRequest(http.MethodGet, "/albums/1", nil)
	router.ServeHTTP(testRecorder, req)
	b, _ := io.ReadAll(testRecorder.Body)
	returnedBody := string(b)

	assert.Equal(t, http.StatusBadRequest, testRecorder.Code)

	finishedSpans := spanRecorder.Ended()
	assert.Len(t, finishedSpans, 1)

	assert.Equal(t, codes.Error, finishedSpans[0].Status().Code)
	assert.Equal(t, "error from album-store getAlbumById malformed JSON", finishedSpans[0].Status().Description)

	assert.Equal(t, 1, len(finishedSpans[0].Events()))
	assert.Equal(t, "error from album-store getAlbumById malformed JSON", finishedSpans[0].Events()[0].Name)

	attributeMap := makeKeyMap(finishedSpans[0].Attributes())
	assert.Equal(t, "400", attributeMap["proxy-service.status_code"].Emit())
	assert.Equal(t, "ID=1", attributeMap["proxy-service.request.parameters"].Emit())
	assert.Equal(t, responseBody, attributeMap["proxy-service.response.body"].Emit())

	assert.Equal(t, `{"message":"error from album-store getAlbumById malformed JSON"}`, returnedBody)
}

func Test_postAlbums_Success(t *testing.T) {
	testRecorder, spanRecorder, router := setupTestRouter()
	DefaultClient = &MockClient{}

	requestBody := `{"artist":"Black Sabbath","id":10,"price":66.6,"title":"The Ozzman Cometh"}`
	requestBodyReader := io.NopCloser(bytes.NewReader([]byte(requestBody)))

	responseBody := `{"artist":"Black Sabbath","id":10,"price":66.6,"title":"The Ozzman Cometh"}`
	responseBodyReader := io.NopCloser(bytes.NewReader([]byte(responseBody)))

	//inject a success message from the server and return a json blob that represents an album
	MockResponseFunc = func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       responseBodyReader,
		}, nil
	}

	req, _ := http.NewRequest(http.MethodPost, "/albums", requestBodyReader)
	router.ServeHTTP(testRecorder, req)
	b, _ := io.ReadAll(testRecorder.Body)
	returnedBody := string(b)

	assert.Equal(t, http.StatusOK, testRecorder.Code)

	finishedSpans := spanRecorder.Ended()
	assert.Len(t, finishedSpans, 1)

	assert.Equal(t, codes.Ok, finishedSpans[0].Status().Code)
	assert.Equal(t, "", finishedSpans[0].Status().Description)

	assert.Equal(t, 0, len(finishedSpans[0].Events()))

	attributeMap := makeKeyMap(finishedSpans[0].Attributes())
	assert.Equal(t, "200", attributeMap["proxy-service.status_code"].Emit())
	assert.Equal(t, requestBody, attributeMap["proxy-service.request.body"].Emit())
	assert.Equal(t, responseBody, attributeMap["proxy-service.response.body"].Emit())

	assert.Equal(t, responseBody, returnedBody)
}

func Test_postAlbums_Failure_Album_Empty_Request_Body(t *testing.T) {
	testRecorder, spanRecorder, router := setupTestRouter()
	DefaultClient = &MockClient{}

	requestBody := ``
	requestBodyReader := io.NopCloser(bytes.NewReader([]byte(requestBody)))

	//Mock not used so setup as ignored
	MockResponseFunc = func(*http.Request) (*http.Response, error) {
		return nil, nil
	}

	req, _ := http.NewRequest(http.MethodPost, "/albums", requestBodyReader)
	router.ServeHTTP(testRecorder, req)
	b, _ := io.ReadAll(testRecorder.Body)
	returnedBody := string(b)

	assert.Equal(t, http.StatusBadRequest, testRecorder.Code)

	finishedSpans := spanRecorder.Ended()
	assert.Len(t, finishedSpans, 1)

	assert.Equal(t, codes.Error, finishedSpans[0].Status().Code)
	assert.Equal(t, "invalid request json body ", finishedSpans[0].Status().Description)

	assert.Equal(t, 1, len(finishedSpans[0].Events()))
	assert.Equal(t, "invalid request json body ", finishedSpans[0].Events()[0].Name)

	attributeMap := makeKeyMap(finishedSpans[0].Attributes())
	assert.Equal(t, "400", attributeMap["proxy-service.status_code"].Emit())
	assert.Equal(t, "", attributeMap["proxy-service.request.body"].Emit())

	assert.Equal(t, `{"message":"invalid request json body "}`, returnedBody)
}

func Test_postAlbums_Failure_Album_Malformed_Request_Body(t *testing.T) {
	testRecorder, spanRecorder, router := setupTestRouter()
	DefaultClient = &MockClient{}

	requestBody := `{"title":"Ozzman Cometh"`
	requestBodyReader := io.NopCloser(bytes.NewReader([]byte(requestBody)))

	//Mock not used so setup as ignored
	MockResponseFunc = func(*http.Request) (*http.Response, error) {
		return nil, nil
	}

	req, _ := http.NewRequest(http.MethodPost, "/albums", requestBodyReader)
	router.ServeHTTP(testRecorder, req)
	b, _ := io.ReadAll(testRecorder.Body)
	returnedBody := string(b)

	assert.Equal(t, http.StatusBadRequest, testRecorder.Code)

	finishedSpans := spanRecorder.Ended()
	assert.Len(t, finishedSpans, 1)

	assert.Equal(t, codes.Error, finishedSpans[0].Status().Code)
	assert.Equal(t, fmt.Sprintf("invalid request json body %v", requestBody), finishedSpans[0].Status().Description)

	assert.Equal(t, 1, len(finishedSpans[0].Events()))
	assert.Equal(t, fmt.Sprintf("invalid request json body %v", requestBody), finishedSpans[0].Events()[0].Name)

	attributeMap := makeKeyMap(finishedSpans[0].Attributes())
	assert.Equal(t, "400", attributeMap["proxy-service.status_code"].Emit())
	assert.Equal(t, requestBody, attributeMap["proxy-service.request.body"].Emit())
	assert.Equal(t, `{"message":"invalid request json body {\"title\":\"Ozzman Cometh\""}`, returnedBody)
	assert.Equal(t, `{"message":"invalid request json body {"title":"Ozzman Cometh""}`, attributeMap["proxy-service.response.body"].Emit())
}

func Test_postAlbums_Failure_Album_Returns_Error(t *testing.T) {
	testRecorder, spanRecorder, router := setupTestRouter()
	DefaultClient = &MockClient{}

	requestBody := `{"artist":"Black Sabbath","id":10,"price":66.6,"title":"The Ozzman Cometh"}`
	requestBodyReader := io.NopCloser(bytes.NewReader([]byte(requestBody)))

	//inject in failure message to respond with that we could not get to the album-store
	MockResponseFunc = func(*http.Request) (*http.Response, error) {
		return nil, errors.New(
			"Error from web server",
		)
	}

	req, _ := http.NewRequest(http.MethodPost, "/albums", requestBodyReader)
	router.ServeHTTP(testRecorder, req)
	b, _ := io.ReadAll(testRecorder.Body)
	returnedBody := string(b)

	assert.Equal(t, http.StatusBadRequest, testRecorder.Code)

	finishedSpans := spanRecorder.Ended()
	assert.Len(t, finishedSpans, 1)

	assert.Equal(t, codes.Error, finishedSpans[0].Status().Code)
	assert.Equal(t, "error contacting album-store postAlbum Error from web server", finishedSpans[0].Status().Description)

	assert.Equal(t, 1, len(finishedSpans[0].Events()))
	assert.Equal(t, "error contacting album-store postAlbum Error from web server", finishedSpans[0].Events()[0].Name)

	attributeMap := makeKeyMap(finishedSpans[0].Attributes())
	assert.Equal(t, "400", attributeMap["proxy-service.status_code"].Emit())
	assert.Equal(t, requestBody, attributeMap["proxy-service.request.body"].Emit())

	assert.Equal(t, `{"message":"error contacting album-store postAlbum Error from web server"}`, returnedBody)
}

func Test_postAlbums_Failure_Malformed_Response(t *testing.T) {
	testRecorder, spanRecorder, router := setupTestRouter()
	DefaultClient = &MockClient{}

	requestBody := `{"artist":"Black Sabbath","id":10,"price":66.6,"title":"The Ozzman Cometh"}`
	requestBodyReader := io.NopCloser(bytes.NewReader([]byte(requestBody)))

	//inject in failure message to respond with that we could not get to the album-store
	responseBody := `[{"artist":"Black Sabbath","id":10,"price":66.6`
	body := io.NopCloser(bytes.NewReader([]byte(responseBody)))

	//inject a failure message from the server and return a json blob that represents an album
	MockResponseFunc = func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: 200,
			Body:       body,
		}, nil
	}

	req, _ := http.NewRequest(http.MethodPost, "/albums", requestBodyReader)
	router.ServeHTTP(testRecorder, req)
	b, _ := io.ReadAll(testRecorder.Body)
	returnedBody := string(b)

	assert.Equal(t, http.StatusBadRequest, testRecorder.Code)

	finishedSpans := spanRecorder.Ended()
	assert.Len(t, finishedSpans, 1)

	assert.Equal(t, codes.Error, finishedSpans[0].Status().Code)
	assert.Equal(t, "error from album-store postAlbum malformed JSON", finishedSpans[0].Status().Description)

	assert.Equal(t, 1, len(finishedSpans[0].Events()))
	assert.Equal(t, "error from album-store postAlbum malformed JSON", finishedSpans[0].Events()[0].Name)

	attributeMap := makeKeyMap(finishedSpans[0].Attributes())
	assert.Equal(t, "400", attributeMap["proxy-service.status_code"].Emit())
	assert.Equal(t, responseBody, attributeMap["proxy-service.response.body"].Emit())
	assert.Equal(t, requestBody, attributeMap["proxy-service.request.body"].Emit())

	assert.Equal(t, `{"message":"error from album-store postAlbum malformed JSON"}`, returnedBody)
}

func TestGet(t *testing.T) {
	type args struct {
		ctx       context.Context
		targetURL string
	}
	tests := []struct {
		name     string
		args     args
		wantResp *http.Response
		wantErr  assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResp, err := Get(tt.args.ctx, tt.args.targetURL)
			if !tt.wantErr(t, err, fmt.Sprintf("Get(%v, %v)", tt.args.ctx, tt.args.targetURL)) {
				return
			}
			assert.Equalf(t, tt.wantResp, gotResp, "Get(%v, %v)", tt.args.ctx, tt.args.targetURL)
		})
	}
}

func TestPost(t *testing.T) {
	type args struct {
		ctx         context.Context
		targetURL   string
		contentType string
		body        io.Reader
	}
	tests := []struct {
		name     string
		args     args
		wantResp *http.Response
		wantErr  assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResp, err := Post(tt.args.ctx, tt.args.targetURL, tt.args.contentType, tt.args.body)
			if !tt.wantErr(t, err, fmt.Sprintf("Post(%v, %v, %v, %v)", tt.args.ctx, tt.args.targetURL, tt.args.contentType, tt.args.body)) {
				return
			}
			assert.Equalf(t, tt.wantResp, gotResp, "Post(%v, %v, %v, %v)", tt.args.ctx, tt.args.targetURL, tt.args.contentType, tt.args.body)
		})
	}
}

func Test_getAlbumByID(t *testing.T) {
	type args struct {
		c *gin.Context
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getAlbumByID(tt.args.c)
		})
	}
}

func Test_getAlbums(t *testing.T) {
	type args struct {
		c *gin.Context
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			getAlbums(tt.args.c)
		})
	}
}

func Test_initOtelProvider(t *testing.T) {
	type args struct {
		serviceName string
		version     string
		gitHash     string
	}
	tests := []struct {
		name    string
		args    args
		want    func(context.Context) error
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := initOtelProvider(tt.args.serviceName, tt.args.version, tt.args.gitHash)
			if !tt.wantErr(t, err, fmt.Sprintf("initOtelProvider(%v, %v, %v)", tt.args.serviceName, tt.args.version, tt.args.gitHash)) {
				return
			}
			assert.Equalf(t, tt.want, got, "initOtelProvider(%v, %v, %v)", tt.args.serviceName, tt.args.version, tt.args.gitHash)
		})
	}
}

func Test_postAlbum(t *testing.T) {
	type args struct {
		c *gin.Context
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postAlbum(tt.args.c)
		})
	}
}

func Test_setupOtelProtoBuffGrpcTrace(t *testing.T) {
	type args struct {
		ctx          context.Context
		otelLocation *string
	}
	tests := []struct {
		name    string
		args    args
		want    *otlptrace.Exporter
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := setupOtelProtoBuffGrpcTrace(tt.args.ctx, tt.args.otelLocation)
			if !tt.wantErr(t, err, fmt.Sprintf("setupOtelProtoBuffGrpcTrace(%v, %v)", tt.args.ctx, tt.args.otelLocation)) {
				return
			}
			assert.Equalf(t, tt.want, got, "setupOtelProtoBuffGrpcTrace(%v, %v)", tt.args.ctx, tt.args.otelLocation)
		})
	}
}

func Test_setupOtelResource(t *testing.T) {
	type args struct {
		serviceName  string
		version      string
		gitHash      string
		ctx          context.Context
		namespace    *string
		instanceName *string
	}
	tests := []struct {
		name    string
		args    args
		want    *resource.Resource
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := setupOtelResource(tt.args.serviceName, tt.args.version, tt.args.gitHash, tt.args.ctx, tt.args.namespace, tt.args.instanceName)
			if !tt.wantErr(t, err, fmt.Sprintf("setupOtelResource(%v, %v, %v, %v, %v, %v)", tt.args.serviceName, tt.args.version, tt.args.gitHash, tt.args.ctx, tt.args.namespace, tt.args.instanceName)) {
				return
			}
			assert.Equalf(t, tt.want, got, "setupOtelResource(%v, %v, %v, %v, %v, %v)", tt.args.serviceName, tt.args.version, tt.args.gitHash, tt.args.ctx, tt.args.namespace, tt.args.instanceName)
		})
	}
}

func Test_setupOtelTraceProvider(t *testing.T) {
	type args struct {
		traceExporter *otlptrace.Exporter
		otelResource  *resource.Resource
	}
	tests := []struct {
		name string
		args args
		want *sdktrace.TracerProvider
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, setupOtelTraceProvider(tt.args.traceExporter, tt.args.otelResource), "setupOtelTraceProvider(%v, %v)", tt.args.traceExporter, tt.args.otelResource)
		})
	}
}

func Test_setupRouter(t *testing.T) {
	tests := []struct {
		name string
		want *gin.Engine
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, setupRouter(), "setupRouter()")
		})
	}
}
