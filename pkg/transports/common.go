package transports

import (
	"encoding/csv"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"project-nm/pkg/configs"
	"project-nm/pkg/contexts"
	"project-nm/pkg/transports/cores" // 確保此處已定義 ErrorResponse

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt"
	"github.com/shopspring/decimal"
)

// AbortAndResponseError 中止並響應錯誤
func AbortAndResponseError(c *gin.Context, statusCode int, message string, err error) {
	resp := cores.NewErrorResponse(statusCode, message, err)
	cores.GenerateGinResponse(c, resp)
	c.Abort()
}

// HandleBearerTokenToUserInfo 解析 JWT 到 UserInfo
func HandleBearerTokenToUserInfo(c *gin.Context) (*contexts.UserInfo, bool) {
	jwtSign := configs.GetConfig().JWTSign
	if val, exists := c.Get("user_info"); exists {
		return val.(*contexts.UserInfo), true
	}

	authHeader := c.Request.Header.Get("Authorization")
	parts := strings.Fields(authHeader)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		AbortAndResponseError(c, http.StatusUnauthorized, "Unauthorized: Invalid Token Format", nil)
		return nil, false
	}

	tokenString := parts[1]
	tokenClaims, err := jwt.ParseWithClaims(
		tokenString,
		&contexts.UserInfo{},
		func(t *jwt.Token) (interface{}, error) {
			return []byte(jwtSign), nil
		},
	)

	if err != nil {
		AbortAndResponseError(c, http.StatusUnauthorized, err.Error(), err)
		return nil, false
	}

	userInfo := tokenClaims.Claims.(*contexts.UserInfo)
	c.Set("user_info", userInfo)
	return userInfo, true
}

// CheckPermissions 檢查權限 
func CheckPermissions(c *gin.Context, requiredPermissions []*cores.Permission) bool {
	userInfo, exists := c.Get("user_info")
	if !exists {
		AbortAndResponseError(c, http.StatusUnauthorized, "Unauthorized", nil)
		return false
	}

	user := userInfo.(*contexts.UserInfo)
	for _, req := range requiredPermissions {
		if !user.Permissions[req.Name] {
			AbortAndResponseError(c, http.StatusForbidden, "Forbidden: No Permission", nil)
			return false
		}
	}
	return true
}

// HandleRequestBody 通用請求體解析與驗證 (支援 Struct 與 Slice)
func HandleRequestBody(c *gin.Context, v interface{}) (interface{}, bool) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	isSlice := t.Kind() == reflect.Slice
	var modelPtr interface{}
	if isSlice {
		modelPtr = reflect.New(reflect.SliceOf(t.Elem())).Interface()
	} else {
		modelPtr = reflect.New(t).Interface()
	}

	if err := c.BindJSON(modelPtr); err != nil {
		AbortAndResponseError(c, http.StatusUnprocessableEntity, err.Error(), err)
		return nil, false
	}

	model := reflect.ValueOf(modelPtr).Elem().Interface()
	validate := validator.New()
	validate.RegisterValidation("no_special_chars", ValidateNoSpecialChars)

	var vErr error
	if isSlice {
		vErr = validate.Var(model, "dive")
	} else {
		vErr = validate.Struct(model)
	}

	if vErr != nil {
		AbortAndResponseError(c, http.StatusBadRequest, vErr.Error(), vErr)
		return nil, false
	}

	return model, true
}

// ValidateNoSpecialChars 自定義驗證：排除特殊 SQL 注入風險字符
func ValidateNoSpecialChars(fl validator.FieldLevel) bool {
	val, ok := fl.Field().Interface().(string)
	if !ok {
		return false
	}
	badChars := []string{"'", "\"", ";", "--", "*", " ", "=", "<", ">"}
	for _, char := range badChars {
		if strings.Contains(val, char) {
			return false
		}
	}
	return true
}

// HandleCsvFile CSV 檔案解析 (針對 project-nm 調整後的邏輯)
func HandleCsvFile(c *gin.Context, v interface{}) (interface{}, bool) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		AbortAndResponseError(c, http.StatusBadRequest, "Missing CSV file", err)
		return nil, false
	}

	file, _ := fileHeader.Open()
	defer file.Close()

	reader := csv.NewReader(file)
	headers, err := reader.Read()
	if err != nil {
		return nil, false
	}

	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[strings.TrimSpace(h)] = i
	}

	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	sliceValue := reflect.MakeSlice(reflect.SliceOf(t), 0, 0)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}

		newItem := reflect.New(t).Elem()
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("csv")
			if idx, ok := headerMap[tag]; ok && idx < len(record) {
				setFieldValue(newItem.Field(i), record[idx])
			}
		}
		sliceValue = reflect.Append(sliceValue, newItem)
	}

	return sliceValue.Interface(), true
}

// setFieldValue 內部輔助方法：處理各類型的轉換 (含 Decimal)
func setFieldValue(field reflect.Value, val string) error {
	val = strings.TrimSpace(val)
	switch field.Kind() {
	case reflect.String:
		field.SetString(val)
	case reflect.Int, reflect.Int64:
		v, _ := strconv.ParseInt(val, 10, 64)
		field.SetInt(v)
	case reflect.Uint, reflect.Uint64:
		v, _ := strconv.ParseUint(val, 10, 64)
		field.SetUint(v)
	case reflect.Float64:
		v, _ := strconv.ParseFloat(val, 64)
		field.SetFloat(v)
	}

	if field.Type().String() == "decimal.Decimal" {
		d, _ := decimal.NewFromString(val)
		field.Set(reflect.ValueOf(d))
	}
	return nil
}
