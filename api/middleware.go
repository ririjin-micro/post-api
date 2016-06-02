package api

import (
	"encoding/json"
	"strings"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

const (
	apiRequestsKey = "apiRequestsKey"
	responseKey    = "apiResponseKey"
)

type APIRequests struct {
	Requests     []PostAPIRequest
	IsMultiCall  bool
	MajorVersion string
}

func (p *PostAPI) cors(next echo.HandlerFunc) echo.HandlerFunc {
	return middleware.CORSWithConfig(
		middleware.CORSConfig{
			AllowOrigins:     p.Options.CORS.AllowOrigins,
			AllowMethods:     p.Options.CORS.AllowMethods,
			AllowHeaders:     p.Options.CORS.AllowHeaders,
			AllowCredentials: p.Options.CORS.AllowCredentials,
			ExposeHeaders:    p.Options.CORS.ExposeHeaders,
			MaxAge:           p.Options.CORS.MaxAge,
		})(next)
}

func (p *PostAPI) writeBasicHeaders(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		if p.Options.ResponseHeader != nil {
			for key, values := range p.Options.ResponseHeader {
				value := strings.Join(values, ";")
				c.Response().Header().Set(key, value)
			}
		}
		return next(c)
	}
}

func (p *PostAPI) parseAPIRequests(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) (err error) {

		var requests *APIRequests
		if requests, err = getAPIRequests(c); err != nil {
			return
		}

		c.Set(apiRequestsKey, requests)

		return next(c)
	}
}

func APIRequestsFromContext(c echo.Context) *APIRequests {
	if !c.Contains(apiRequestsKey) {
		return nil
	}

	v := c.Get(apiRequestsKey)
	if v == nil {
		return nil
	}

	if requests, ok := v.(*APIRequests); ok {
		return requests
	}

	return nil
}

func APIResponsesFromContext(c echo.Context) map[string]PostAPIResponse {
	if !c.Contains(responseKey) {
		return nil
	}

	v := c.Get(responseKey)
	if v == nil {
		return nil
	}

	if responses, ok := v.(map[string]PostAPIResponse); ok {
		return responses
	}

	return nil
}

func getAPIRequests(c echo.Context) (apiRequests *APIRequests, err error) {
	multiCall := false
	mcVal := strings.ToLower(c.Request().Header().Get(MultiCallHeader))
	if mcVal != "" {
		if mcVal == "on" ||
			mcVal == "1" ||
			mcVal == "true" {
			multiCall = true
		}
	}

	apiVersion := "v1"
	requestVer := c.Param("version")
	if requestVer != "" {
		apiVersion = requestVer
	}

	var requests []PostAPIRequest

	// multi api calls
	if multiCall {

		var multiRequest map[string]map[string]interface{}

		decoder := json.NewDecoder(c.Request().Body())
		decoder.UseNumber()
		if err = decoder.Decode(&multiRequest); err != nil {
			return
		}

		if multiRequest != nil {
			for tmpAPI, request := range multiRequest {

				api := ""
				ver := apiVersion
				isSpecificVersion := false

				apiV := strings.Split(tmpAPI, ":")
				if len(apiV) == 2 {
					ver = strings.TrimSpace(apiV[1])
					isSpecificVersion = true
				}

				api = strings.TrimSpace(apiV[0])

				if api == "" {
					err = ErrBadRequest.New().Append("API name is empty")
					return
				}

				requests = append(requests,
					PostAPIRequest{
						API:               api,
						Content:           request,
						Version:           ver,
						IsSpecificVersion: isSpecificVersion,
					},
				)
			}
		}

		apiRequests = &APIRequests{
			Requests:     requests,
			IsMultiCall:  true,
			MajorVersion: requestVer,
		}

		return
	}

	// singal api call
	api := c.Request().Header().Get(APIHeader)
	api = strings.TrimSpace(api)
	if api == "" {
		err = ErrBadRequest.New().Append("API name is empty")
		return
	}

	var request map[string]interface{}
	decoder := json.NewDecoder(c.Request().Body())
	decoder.UseNumber()
	if err = decoder.Decode(&request); err != nil {
		return
	}

	requests = append(requests, PostAPIRequest{API: api, Content: request, Version: apiVersion})

	apiRequests = &APIRequests{
		Requests:     requests,
		IsMultiCall:  false,
		MajorVersion: requestVer,
	}

	return
}
