package errors

// func TestInit(t *testing.T) {
// 	box, err := rice.FindBox("../../errors")
// 	assert.NoError(t, err)
// 	Init(box)
// }

// func TestNew(t *testing.T) {
// 	e := New("E0001")
// 	assert.Error(t, e)
// 	code, err := HTTPResponse(e)
// 	assert.NotNil(t, err)
// 	assert.Equal(t, http.StatusInternalServerError, code)
// }

// func TestHTTPResponse(t *testing.T) {
// 	e := New("E0001")
// 	assert.Error(t, e)
// 	code, err := e.HTTPResponse()
// 	assert.NotNil(t, err)
// 	assert.Equal(t, http.StatusInternalServerError, code)
// }

// func TestNewForExceptionalError(t *testing.T) {
// 	e := New("E1001")
// 	assert.Error(t, e)
// }

// func TestNewForSystemErrors(t *testing.T) {
// 	code, e := HTTPResponse(errors.New("Test"))
// 	assert.NotNil(t, e)
// 	assert.Equal(t, http.StatusInternalServerError, code)

// }
