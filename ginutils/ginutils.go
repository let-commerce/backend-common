package ginutils

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"github.com/let-commerce/backend-common/response"
	"github.com/let-commerce/backend-common/utils/datetime"
	"github.com/pkg/errors"
	"net/http"
	"strconv"
	"time"
)

type GinUtils struct {
	ctx *gin.Context
}

func Init(ginCtx *gin.Context) *GinUtils {
	return &GinUtils{ginCtx}
}

// GetIntParam method binds new int Param from ctx and return http.StatusBadRequest if it couldn't parse
func (utils *GinUtils) GetIntParam(paramName string) (int, error) {
	paramVal := utils.ctx.Params.ByName(paramName)
	intParam, err := strconv.Atoi(paramVal)
	if err != nil {
		utils.ctx.JSON(http.StatusBadRequest, response.NewErrorResponseF(errors.WithStack(err), "can't bind param: %v to int (value = %v)", paramName, paramVal))
	}
	return intParam, err
}

// GetUIntParam method binds new uint Param from ctx and return http.StatusBadRequest if it couldn't parse
func (utils *GinUtils) GetUIntParam(paramName string) (uint, error) {
	paramVal := utils.ctx.Params.ByName(paramName)
	intParam, err := strconv.Atoi(paramVal)
	if err != nil {
		utils.ctx.JSON(http.StatusBadRequest, response.NewErrorResponseF(errors.WithStack(err), "can't bind param: %v to uint (value = %v)", paramName, paramVal))
	}
	return uint(intParam), err
}

// GetIntQuery method binds new int Param from ctx query and return http.StatusBadRequest if it couldn't parse
func (utils *GinUtils) GetIntQuery(paramName string) (int, bool, error) {
	paramVal, exists := utils.ctx.GetQuery(paramName)
	if exists && paramVal != "null" {
		intParam, err := strconv.Atoi(paramVal)
		if err != nil {
			utils.ctx.JSON(http.StatusBadRequest, response.NewErrorResponseF(errors.WithStack(err), "can't bind param: %v to int (value = %v)", paramName, paramVal))
		}
		return intParam, exists, err
	}
	return 0, exists, nil
}

// GetUIntQuery method binds new uint Param from ctx query and return http.StatusBadRequest if it couldn't parse
func (utils *GinUtils) GetUIntQuery(paramName string) (uint, bool, error) {
	paramVal, exists := utils.ctx.GetQuery(paramName)
	if exists && paramVal != "null" {
		intParam, err := strconv.Atoi(paramVal)
		if err != nil {
			utils.ctx.JSON(http.StatusBadRequest, response.NewErrorResponseF(errors.WithStack(err), "can't bind param: %v to uint (value = %v)", paramName, paramVal))
		}
		return uint(intParam), exists, err
	}
	return 0, exists, nil
}

// GetDateParam method binds date string Param from ctx (in format: "2006-01-15")
func (utils *GinUtils) GetDateParam(paramName string) (time.Time, error) {
	paramVal := utils.ctx.Params.ByName(paramName)
	date, err := datetime.ParseDate(paramVal)
	return date, err
}

// GetDateQuery method binds date string Param from ctx (in format: "2006-01-15")
func (utils *GinUtils) GetDateQuery(paramName string) (time.Time, bool, error) {
	paramVal, exists := utils.ctx.GetQuery(paramName)
	if exists && paramVal != "null" {
		date, err := datetime.ParseDate(paramVal)
		return date, exists, err
	}
	return time.Time{}, exists, nil
}

// GetStringParam method binds new string Param from ctx
func (utils *GinUtils) GetStringParam(paramName string) string {
	paramVal := utils.ctx.Params.ByName(paramName)
	return paramVal
}

// GetStringQuery method binds new string Query from ctx
func (utils *GinUtils) GetStringQuery(paramName string) (string, bool) {
	paramVal, exists := utils.ctx.GetQuery(paramName)
	if paramVal == "null" {
		return "", false
	}
	return paramVal, exists
}

// GetBoolQuery method binds bool string Param from ctx query (example: "true")
func (utils *GinUtils) GetBoolQuery(paramName string) (bool, bool, error) {
	paramVal, exists := utils.ctx.GetQuery(paramName)
	if exists && paramVal != "null" {
		result, err := strconv.ParseBool(paramVal)
		return result, exists, err
	}
	return false, exists, nil
}

// BindDTO method binds new DTO from ctx body
func BindDTO[T any](ctx *gin.Context, dto T) (T, error) {
	err := ctx.Bind(&dto)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewErrorResponse(fmt.Sprintf("Got error while binding dto: %v", dto), errors.WithStack(err)))
	}
	return dto, err
}

// BindMap method binds new map from ctx body
func (utils *GinUtils) BindMap() (map[string]interface{}, error) {
	var dto map[string]interface{}
	err := utils.ctx.Bind(&dto)
	if err != nil {
		utils.ctx.JSON(http.StatusInternalServerError, response.NewErrorResponse("Got error while binding map", errors.WithStack(err)))
	}
	return dto, err
}

type IValidatable interface {
	Validate() error
}

// BindAndValidateDTO method binds new DTO from ctx body
func BindAndValidateDTO[T IValidatable](ctx *gin.Context, dto T) (T, error) {
	var null T
	err := ctx.Bind(&dto)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewErrorResponse("Got error while binding dto", errors.WithStack(err)))
		return null, err
	}
	err = dto.Validate()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, response.NewErrorResponse("Got error while validating dto", errors.WithStack(err)))
		return null, err
	}
	return dto, err
}

func (utils *GinUtils) ReturnResultOrError(result interface{}, errMessage string, err error) {
	if err == nil {
		utils.ctx.JSON(http.StatusOK, result)
	} else {
		utils.ctx.JSON(http.StatusInternalServerError, response.NewErrorResponse(errMessage, errors.WithStack(err)))
	}
}

func (utils *GinUtils) ReturnInternalServerError(errMessage string, err error) {
	utils.ctx.JSON(http.StatusInternalServerError, response.NewErrorResponse(errMessage, errors.WithStack(err)))
}

func (utils *GinUtils) ReturnBadRequestError(errMessage string, err error) {
	utils.ctx.JSON(http.StatusBadRequest, response.NewErrorResponse(errMessage, errors.WithStack(err)))
}

func (utils *GinUtils) ReturnMessageResponseOrError(message string, errMessage string, err error) {
	if err == nil {
		utils.ctx.JSON(http.StatusOK, response.Response{Message: message})
	} else {
		utils.ctx.JSON(http.StatusInternalServerError, response.NewErrorResponse(errMessage, errors.WithStack(err)))
	}
}

func (utils *GinUtils) ReturnMessageResponseWithIdOrError(message string, id uint, errMessage string, err error) {
	if err == nil {
		utils.ctx.JSON(http.StatusOK, response.Response{Message: message, ID: id})
	} else {
		utils.ctx.JSON(http.StatusInternalServerError, response.NewErrorResponse(errMessage, errors.WithStack(err)))
	}
}

func CopyDTO[T any](ctx *gin.Context, to T, from interface{}, ignoreEmpty bool) (result T, err error) {
	var null T
	if ignoreEmpty {
		err = copier.CopyWithOption(&to, from, copier.Option{IgnoreEmpty: true})
	} else {
		err = copier.Copy(&to, from)
	}
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, response.NewErrorResponse("Got error while coping", errors.WithStack(err)))
		return null, err
	}
	return to, nil
}
