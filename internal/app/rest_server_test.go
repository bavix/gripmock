package app_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gripmock/stuber"
	"github.com/stretchr/testify/require"

	"github.com/bavix/features"
	"github.com/bavix/gripmock/internal/app"
)

//nolint:all
func TestStub(t *testing.T) {
	type test struct {
		name    string
		mock    func() *http.Request
		handler http.HandlerFunc
		expect  string
	}

	api, _ := app.NewRestServer(
		stuber.NewBudgerigar(features.New()),
		nil,
		nil,
	)

	//nolint:lll
	cases := []test{
		{
			name: "add simple stub",
			mock: func() *http.Request {
				payload := `{
						"id": "43739ed8-2810-4f57-889b-4d3ff5795bce",
						"service": "Testing",
						"method":"TestMethod",
						"input":{
							"equals":{
								"Hola":"Mundo"
							}
						},
						"output":{
							"data":{
								"Hello":"World"
							}
						}
					}`
				read := bytes.NewReader([]byte(payload))

				return httptest.NewRequest(http.MethodPost, "/api/stubs", read)
			},
			handler: api.AddStub,
			expect:  `["43739ed8-2810-4f57-889b-4d3ff5795bce"]`,
		},
		{
			name: "list stub",
			mock: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/stubs", nil)
			},
			handler: api.ListStubs,
			expect:  "[{\"id\":\"43739ed8-2810-4f57-889b-4d3ff5795bce\",\"service\":\"Testing\",\"method\":\"TestMethod\",\"headers\":{\"equals\":null,\"contains\":null,\"matches\":null},\"input\":{\"equals\":{\"Hola\":\"Mundo\"},\"contains\":null,\"matches\":null},\"output\":{\"data\":{\"Hello\":\"World\"},\"error\":\"\",\"headers\":null}}]",
		},
		{
			name: "unused stubs (all stubs)",
			mock: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/stubs/unused", nil)
			},
			handler: api.ListUnusedStubs,
			expect:  "[{\"id\":\"43739ed8-2810-4f57-889b-4d3ff5795bce\",\"service\":\"Testing\",\"method\":\"TestMethod\",\"headers\":{\"equals\":null,\"contains\":null,\"matches\":null},\"input\":{\"equals\":{\"Hola\":\"Mundo\"},\"contains\":null,\"matches\":null},\"output\":{\"data\":{\"Hello\":\"World\"},\"error\":\"\",\"headers\":null}}]",
		},
		{
			name: "used stubs (zero)",
			mock: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/stubs/used", nil)
			},
			handler: api.ListUsedStubs,
			expect:  "[]",
		},
		{
			name: "find stub equals using features.RequestInternal",
			mock: func() *http.Request {
				payload := `{"service":"Testing","method":"TestMethod","data":{"Hola":"Mundo"}}`

				req := httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewReader([]byte(payload)))
				req.Header.Add(strings.ToUpper("X-GripMock-RequestInternal"), "ok") // enable

				return req
			},
			handler: api.SearchStubs,
			expect:  "{\"data\":{\"Hello\":\"World\"},\"error\":\"\",\"headers\":null}\n",
		},
		{
			name: "used stubs (zero)",
			mock: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/stubs/used", nil)
			},
			handler: api.ListUsedStubs,
			expect:  "[]",
		},
		{
			name: "find stub equals",
			mock: func() *http.Request {
				payload := `{"service":"Testing","method":"TestMethod","data":{"Hola":"Mundo"}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			handler: api.SearchStubs,
			expect:  "{\"data\":{\"Hello\":\"World\"},\"error\":\"\",\"headers\":null}\n",
		},
		{
			name: "unused stubs (zero)",
			mock: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/stubs/unused", nil)
			},
			handler: api.ListUnusedStubs,
			expect:  "[]",
		},
		{
			name: "used stubs (all stubs)",
			mock: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/stubs/used", nil)
			},
			handler: api.ListUsedStubs,
			expect:  "[{\"id\":\"43739ed8-2810-4f57-889b-4d3ff5795bce\",\"service\":\"Testing\",\"method\":\"TestMethod\",\"headers\":{\"equals\":null,\"contains\":null,\"matches\":null},\"input\":{\"equals\":{\"Hola\":\"Mundo\"},\"contains\":null,\"matches\":null},\"output\":{\"data\":{\"Hello\":\"World\"},\"error\":\"\",\"headers\":null}}]",
		},
		{
			name: "find stub by ID",
			mock: func() *http.Request {
				payload := `{"id": "43739ed8-2810-4f57-889b-4d3ff5795bce", "service":"Testing","method":"TestMethod","data":{}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			handler: api.SearchStubs,
			expect:  "{\"data\":{\"Hello\":\"World\"},\"error\":\"\",\"headers\":null}\n",
		},
		{
			name: "add nested stub equals",
			mock: func() *http.Request {
				payload := `{
						"id": "b7211be4-06f7-4a2c-8453-359f077bcdb8",
						"service": "NestedTesting",
						"method":"TestMethod",
						"input":{
							"equals":{
										"name": "Afra Gokce",
										"age": 1,
										"girl": true,
										"null": null,
										"greetings": {
											"hola": "mundo",
											"merhaba": "dunya"
										},
										"cities": ["Istanbul", "Jakarta"]
							}
						},
						"output":{
							"data":{
								"Hello":"World"
							}
						}
					}`
				read := bytes.NewReader([]byte(payload))

				return httptest.NewRequest(http.MethodPost, "/api/stubs", read)
			},
			handler: api.AddStub,
			expect:  `["b7211be4-06f7-4a2c-8453-359f077bcdb8"]`,
		},
		{
			name: "find nested stub equals",
			mock: func() *http.Request {
				payload := `{"service":"NestedTesting","method":"TestMethod","data":{"name":"Afra Gokce","age":1,"girl":true,"null":null,"greetings":{"hola":"mundo","merhaba":"dunya"},"cities":["Istanbul","Jakarta"]}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			handler: api.SearchStubs,
			expect:  "{\"data\":{\"Hello\":\"World\"},\"error\":\"\",\"headers\":null}\n",
		},
		{
			name: "add stub contains",
			mock: func() *http.Request {
				payload := `{
								"id": "b5e35447-45bb-4b71-8ab4-41ba5dda669c",
								"service": "Testing",
								"method":"TestMethod",
								"input":{
									"contains":{
										"field1":"hello field1",
										"field3":"hello field3"
									}
								},
								"output":{
									"data":{
										"hello":"world"
									}
								}
							}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader([]byte(payload)))
			},
			handler: api.AddStub,
			expect:  `["b5e35447-45bb-4b71-8ab4-41ba5dda669c"]`,
		},
		{
			name: "find stub contains",
			mock: func() *http.Request {
				payload := `{
						"service":"Testing",
						"method":"TestMethod",
						"data":{
							"field1":"hello field1",
							"field2":"hello field2",
							"field3":"hello field3"
						}
					}`

				return httptest.NewRequest(http.MethodGet, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			handler: api.SearchStubs,
			expect:  "{\"data\":{\"hello\":\"world\"},\"error\":\"\",\"headers\":null}\n",
		},
		{
			name: "add nested stub contains",
			mock: func() *http.Request {
				payload := `{
								"id": "b8e354d9-a211-49c7-9947-b617e1689e0f",
								"service": "NestedTesting",
								"method":"TestMethod",
								"input":{
									"contains":{
												"key": "value",
												"greetings": {
													"hola": "mundo",
													"merhaba": "dunya"
												},
												"cities": ["Istanbul", "Jakarta"]
									}
								},
								"output":{
									"data":{
										"hello":"world"
									}
								}
							}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader([]byte(payload)))
			},
			handler: api.AddStub,
			expect:  `["b8e354d9-a211-49c7-9947-b617e1689e0f"]`,
		},
		{
			name: "add multi stub contains",
			mock: func() *http.Request {
				payload := `[{
								"id": "3f68f410-bb58-49ad-b679-23f2ed690c1d",
								"service": "NestedTesting",
								"method":"TestMethod",
								"input":{
									"equals":{
												"key": "stab1",
												"greetings": {
													"hola": "mundo",
													"merhaba": "dunya"
												},
												"cities": ["Istanbul", "Jakarta"]
									}
								},
								"output":{
									"data":{
										"hello":"world"
									}
								}
							},{
								"id": "6da11d72-c0db-4075-9e72-31d61ffd0483",
								"service": "NestedTesting",
								"method":"TestMethod",
								"input":{
									"equals":{
												"key": "stab2",
												"greetings": {
													"hola": "mundo",
													"merhaba": "dunya"
												},
												"cities": ["Istanbul", "Jakarta"]
									}
								},
								"output":{
									"data":{
										"hello":"world"
									}
								}
							}]`

				return httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader([]byte(payload)))
			},
			handler: api.AddStub,
			expect:  `["3f68f410-bb58-49ad-b679-23f2ed690c1d","6da11d72-c0db-4075-9e72-31d61ffd0483"]`,
		},
		{
			name: "add error stub with result code contains",
			mock: func() *http.Request {
				payload := `{
								"id": "cda7321b-9241-4a58-9cbf-0603e0146542",
								"service": "ErrorStabWithCode",
								"method":"TestMethod",
								"input":{
									"contains":{
												"key": "value",
												"greetings": {
													"hola": "mundo",
													"merhaba": "dunya"
												},
												"cities": ["Istanbul", "Jakarta"]
									}
								},
								"output":{
									"error":"error msg",
                                    "code": 3
								}
							}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader([]byte(payload)))
			},
			handler: api.AddStub,
			expect:  `["cda7321b-9241-4a58-9cbf-0603e0146542"]`,
		},
		{
			name: "find error stub with result code contains",
			mock: func() *http.Request {
				payload := `{
						"service": "ErrorStabWithCode",
						"method":"TestMethod",
						"data":{
								"key": "value",
								"anotherKey": "anotherValue",
								"greetings": {
									"hola": "mundo",
									"merhaba": "dunya",
									"hello": "world"
								},
								"cities": ["Istanbul", "Jakarta", "Winterfell"]
						}
					}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			handler: api.SearchStubs,
			expect:  "{\"data\":null,\"error\":\"error msg\",\"code\":3,\"headers\":null}\n",
		},
		{
			name: "add error stub without result code contains",
			mock: func() *http.Request {
				payload := `{
								"id": "6d5ec9a6-94a7-4f23-b5ea-b04a37796adb",
								"service": "ErrorStab",
								"method":"TestMethod",
								"input":{
									"contains":{
												"key": "value",
												"greetings": {
													"hola": "mundo",
													"merhaba": "dunya"
												},
												"cities": ["Istanbul", "Jakarta"]
									}
								},
								"output":{
									"error":"error msg"
								}
							}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader([]byte(payload)))
			},
			handler: api.AddStub,
			expect:  `["6d5ec9a6-94a7-4f23-b5ea-b04a37796adb"]`,
		},
		{
			name: "find error stub without result code contains",
			mock: func() *http.Request {
				payload := `{
						"service": "ErrorStab",
						"method":"TestMethod",
						"data":{
								"key": "value",
								"anotherKey": "anotherValue",
								"greetings": {
									"hola": "mundo",
									"merhaba": "dunya",
									"hello": "world"
								},
								"cities": ["Istanbul", "Jakarta", "Winterfell"]
						}
					}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			handler: api.SearchStubs,
			expect:  "{\"data\":null,\"error\":\"error msg\",\"headers\":null}\n",
		},
		{
			name: "find nested stub contains",
			mock: func() *http.Request {
				payload := `{
						"service":"NestedTesting",
						"method":"TestMethod",
						"data":{
								"key": "value",
								"anotherKey": "anotherValue",
								"greetings": {
									"hola": "mundo",
									"merhaba": "dunya",
									"hello": "world"
								},
								"cities": ["Istanbul", "Jakarta", "Winterfell"]
						}
					}`

				return httptest.NewRequest(http.MethodGet, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			handler: api.SearchStubs,
			expect:  "{\"data\":{\"hello\":\"world\"},\"error\":\"\",\"headers\":null}\n",
		},
		{
			name: "add stub matches regex",
			mock: func() *http.Request {
				payload := `{
						"id": "faf39edb-c695-493f-a25e-ecfc171977dc",
						"service":"Testing2",
						"method":"TestMethod",
						"input":{
							"matches":{
								"field1":".*ello$"
							}
						},
						"output":{
							"data":{
								"reply":"OK"
							}
						}
					}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader([]byte(payload)))
			},
			handler: api.AddStub,
			expect:  `["faf39edb-c695-493f-a25e-ecfc171977dc"]`,
		},
		{
			name: "find stub matches regex",
			mock: func() *http.Request {
				payload := `{
						"service":"Testing2",
						"method":"TestMethod",
						"data":{
							"field1":"hello"
						}
					}`

				return httptest.NewRequest(http.MethodGet, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			handler: api.SearchStubs,
			expect:  "{\"data\":{\"reply\":\"OK\"},\"error\":\"\",\"headers\":null}\n",
		},
		{
			name: "add nested stub matches regex",
			mock: func() *http.Request {
				payload := `{
						"id": "b1299ce3-a2a6-4fe7-94d4-0b68fc80afaa",
						"service":"NestedTesting2",
						"method":"TestMethod",
						"input":{
							"matches":{
										"key": "[a-z]{3}ue",
										"greetings": {
											"hola": 1,
											"merhaba": true,
											"hello": "^he[l]{2,}o$"
										},
										"cities": ["Istanbul", "Jakarta", ".*"],
										"mixed": [5.5, false, ".*"]
							}
						},
						"output":{
							"data":{
								"reply":"OK"
							}
						}
					}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader([]byte(payload)))
			},
			handler: api.AddStub,
			expect:  `["b1299ce3-a2a6-4fe7-94d4-0b68fc80afaa"]`,
		},
		{
			name: "find nested stub matches regex",
			mock: func() *http.Request {
				payload := `{
						"service":"NestedTesting2",
						"method":"TestMethod",
						"data":{
								"key": "value",
								"greetings": {
									"hola": 1,
									"merhaba": true,
									"hello": "helllllo"
								},
								"cities": ["Istanbul", "Jakarta", "Gotham"],
								"mixed": [5.5, false, "Gotham"]
							}
						}
					}`

				return httptest.NewRequest(http.MethodGet, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			handler: api.SearchStubs,
			expect:  "{\"data\":{\"reply\":\"OK\"},\"error\":\"\",\"headers\":null}\n",
		},
		{
			name: "error find stub contains",
			mock: func() *http.Request {
				payload := `{
						"service":"Testing",
						"method":"TestMethod",
						"data":{
							"field1":"hello field1"
						}
					}`

				return httptest.NewRequest(http.MethodGet, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			handler: api.SearchStubs,
			expect:  "{\"error\":\"Can't find stub \\n\\nService: Testing \\n\\nMethod: TestMethod \\n\\nInput\\n\\n{\\n\\t\\\"field1\\\": \\\"hello field1\\\"\\n}\\n\\nClosest Match \\n\\ncontains:{\\n\\t\\\"field1\\\": \\\"hello field1\\\",\\n\\t\\\"field3\\\": \\\"hello field3\\\"\\n}\"}",
		},
		{
			name: "error find stub equals",
			mock: func() *http.Request {
				payload := `{"service":"Testing","method":"TestMethod","data":{"Hola":"Dunia"}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			handler: api.SearchStubs,
			expect:  "{\"error\":\"Can't find stub \\n\\nService: Testing \\n\\nMethod: TestMethod \\n\\nInput\\n\\n{\\n\\t\\\"Hola\\\": \\\"Dunia\\\"\\n}\\n\\nClosest Match \\n\\nequals:{\\n\\t\\\"Hola\\\": \\\"Mundo\\\"\\n}\"}",
		},
	}

	for _, v := range cases {
		t.Run(v.name, func(t *testing.T) {
			wrt := httptest.NewRecorder()
			req := v.mock()
			v.handler(wrt, req)
			res, err := io.ReadAll(wrt.Result().Body)

			require.NoError(t, err)
			require.JSONEq(t, v.expect, string(res), string(res))
		})
	}

	t.Run("purge handler", func(t *testing.T) {
		deleteWrt := httptest.NewRecorder()
		deleteReq := httptest.NewRequest(http.MethodDelete, "/api/stubs", nil)

		api.PurgeStubs(deleteWrt, deleteReq)

		listWrt := httptest.NewRecorder()
		listReq := httptest.NewRequest(http.MethodGet, "/api/stubs", nil)

		api.ListStubs(listWrt, listReq)

		res, err := io.ReadAll(listWrt.Result().Body)

		require.NoError(t, listWrt.Result().Body.Close())

		require.NoError(t, err)
		require.Equal(t, http.StatusNoContent, deleteWrt.Result().StatusCode)

		require.NoError(t, deleteWrt.Result().Body.Close())

		require.JSONEq(t, "[]", string(res))
	})
}
