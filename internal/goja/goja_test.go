package goja_test

// func TestPromiseResolve(t *testing.T) {
// 	is := is.New(t)
// 	log := logs.Default()
// 	vm := goja.New(log)
// 	props := map[string]any{}
// 	actual := new(bytes.Buffer)
// 	err := vm.Render(actual, "test.js", `
// 		new Promise((resolve, reject) => {
// 			resolve("hello")
// 		})
// 	`, props)
// 	is.NoErr(err)
// 	is.Equal(actual.String(), "hello")
// }

// func TestPromiseReject(t *testing.T) {
// 	is := is.New(t)
// 	log := logs.Default()
// 	vm := goja.New(log)
// 	props := map[string]any{}
// 	actual := new(bytes.Buffer)
// 	err := vm.Render(actual, "test.js", `
// 		new Promise((resolve, reject) => {
// 			reject("hello")
// 		})
// 	`, props)
// 	is.True(err != nil)
// 	is.Equal(err.Error(), `goja: promise was rejected with "hello"`)
// 	is.Equal(actual.String(), "")
// }

// func TestURLPathname(t *testing.T) {
// 	is := is.New(t)
// 	log := logs.Default()
// 	vm := goja.New(log)
// 	props := map[string]any{}
// 	actual := new(bytes.Buffer)
// 	err := vm.Render(actual, "test.js", `
// 		const url = new URL("https://sub.example.com:8080/p/a/t/h?query=string#hash")
// 		url.pathname
// 	`, props)
// 	is.NoErr(err)
// 	is.Equal(actual.String(), "/p/a/t/h")
// }
