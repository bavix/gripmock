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
	"github.com/bavix/gripmock/v3/internal/app"
)

//nolint:funlen,maintidx
func TestStub(t *testing.T) {
	type scenario struct {
		description  string
		setupRequest func() *http.Request
		endpoint     http.HandlerFunc
		expected     string
	}

	server, _ := app.NewRestServer(
		t.Context(),
		stuber.NewBudgerigar(features.New()),
		nil,
	)

	testScenarios := []scenario{
		{
			description: "create_basic_stub",
			setupRequest: func() *http.Request {
				payload := `{"id":"43739ed8-2810-4f57-889b-4d3ff5795bce","service":"MammalService","method":"CheckHabitat","input":{"equals":{"Hola":"Mundo"}},"output":{"data":{"Hello":"World"}}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.AddStub,
			expected: `["43739ed8-2810-4f57-889b-4d3ff5795bce"]`,
		},
		{
			description: "retrieve_all_stubs",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/stubs", nil)
			},
			endpoint: server.ListStubs,
			expected: `[{"id":"43739ed8-2810-4f57-889b-4d3ff5795bce","service":"MammalService","method":"CheckHabitat","headers":{"equals":null,"contains":null,"matches":null},"input":{"equals":{"Hola":"Mundo"},"contains":null,"matches":null},"output":{"data":{"Hello":"World"},"error":"","headers":null},"priority":0,"stream":null}]`,
		},
		{
			description: "list_inactive_stubs",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/stubs/unused", nil)
			},
			endpoint: server.ListUnusedStubs,
			expected: `[{"id":"43739ed8-2810-4f57-889b-4d3ff5795bce","service":"MammalService","method":"CheckHabitat","headers":{"equals":null,"contains":null,"matches":null},"input":{"equals":{"Hola":"Mundo"},"contains":null,"matches":null},"output":{"data":{"Hello":"World"},"error":"","headers":null},"priority":0,"stream":null}]`,
		},
		{
			description: "check_empty_active_stubs",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/stubs/used", nil)
			},
			endpoint: server.ListUsedStubs,
			expected: "[]",
		},
		{
			description: "locate_stub_with_internal_flag",
			setupRequest: func() *http.Request {
				payload := `{"service":"MammalService","method":"CheckHabitat","data":{"Hola":"Mundo"}}`
				req := httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewReader([]byte(payload)))
				req.Header.Add(strings.ToUpper("X-GripMock-RequestInternal"), "enabled")

				return req
			},
			endpoint: server.SearchStubs,
			expected: `{"data":{"Hello":"World"},"error":"","headers":null}` + "\n",
		},
		{
			description: "verify_empty_active_stubs_again",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/stubs/used", nil)
			},
			endpoint: server.ListUsedStubs,
			expected: "[]",
		},
		{
			description: "locate_stub_without_internal_flag",
			setupRequest: func() *http.Request {
				payload := `{"service":"MammalService","method":"CheckHabitat","data":{"Hola":"Mundo"}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.SearchStubs,
			expected: `{"data":{"Hello":"World"},"error":"","headers":null}` + "\n",
		},
		{
			description: "confirm_no_inactive_stubs",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/stubs/unused", nil)
			},
			endpoint: server.ListUnusedStubs,
			expected: "[]",
		},
		{
			description: "check_full_active_stubs",
			setupRequest: func() *http.Request {
				return httptest.NewRequest(http.MethodGet, "/api/stubs/used", nil)
			},
			endpoint: server.ListUsedStubs,
			expected: `[{"id":"43739ed8-2810-4f57-889b-4d3ff5795bce","service":"MammalService","method":"CheckHabitat","headers":{"equals":null,"contains":null,"matches":null},"input":{"equals":{"Hola":"Mundo"},"contains":null,"matches":null},"output":{"data":{"Hello":"World"},"error":"","headers":null},"priority":0,"stream":null}]`,
		},
		{
			description: "find_stub_by_identifier",
			setupRequest: func() *http.Request {
				payload := `{"id":"43739ed8-2810-4f57-889b-4d3ff5795bce","service":"MammalService","method":"CheckHabitat","data":{}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.SearchStubs,
			expected: `{"data":{"Hello":"World"},"error":"","headers":null}` + "\n",
		},
		{
			description: "add_complex_stub",
			setupRequest: func() *http.Request {
				payload := `{"id":"b7211be4-06f7-4a2c-8453-359f077bcdb8","service":"ReptileService","method":"ValidateTraits","input":{"equals":{"name":"Afra Gokce","age":1,"girl":true,"null":null,"greetings":{"hola":"mundo","merhaba":"dunya"},"cities":["Istanbul","Jakarta"]}},"output":{"data":{"Hello":"World"}}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.AddStub,
			expected: `["b7211be4-06f7-4a2c-8453-359f077bcdb8"]`,
		},
		{
			description: "match_complex_stub",
			setupRequest: func() *http.Request {
				payload := `{"service":"ReptileService","method":"ValidateTraits","data":{"name":"Afra Gokce","age":1,"girl":true,"null":null,"greetings":{"hola":"mundo","merhaba":"dunya"},"cities":["Istanbul","Jakarta"]}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.SearchStubs,
			expected: `{"data":{"Hello":"World"},"error":"","headers":null}` + "\n",
		},
		{
			description: "create_partial_match_stub",
			setupRequest: func() *http.Request {
				payload := `{"id":"b5e35447-45bb-4b71-8ab4-41ba5dda669c","service":"AmphibianService","method":"CheckMetamorphosis","input":{"contains":{"field1":"hello field1","field3":"hello field3"}},"output":{"data":{"hello":"world"}}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.AddStub,
			expected: `["b5e35447-45bb-4b71-8ab4-41ba5dda669c"]`,
		},
		{
			description: "locate_partial_match_stub",
			setupRequest: func() *http.Request {
				payload := `{"service":"AmphibianService","method":"CheckMetamorphosis","data":{"field1":"hello field1","field2":"hello field2","field3":"hello field3"}}`

				return httptest.NewRequest(http.MethodGet, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.SearchStubs,
			expected: `{"data":{"hello":"world"},"error":"","headers":null}` + "\n",
		},
		{
			description: "add_nested_partial_stub",
			setupRequest: func() *http.Request {
				payload := `{"id":"b8e354d9-a211-49c7-9947-b617e1689e0f","service":"BirdService","method":"TrackMigration","input":{"contains":{"key":"value","greetings":{"hola":"mundo","merhaba":"dunya"},"cities":["Istanbul","Jakarta"]}},"output":{"data":{"hello":"world"}}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.AddStub,
			expected: `["b8e354d9-a211-49c7-9947-b617e1689e0f"]`,
		},
		{
			description: "batch_stub_creation",
			setupRequest: func() *http.Request {
				payload := `[{"id":"3f68f410-bb58-49ad-b679-23f2ed690c1d","service":"FishService","method":"AnalyzeSwarm","input":{"equals":{"key":"stab1","greetings":{"hola":"mundo","merhaba":"dunya"},"cities":["Istanbul","Jakarta"]}},"output":{"data":{"hello":"world"}}},{"id":"6da11d72-c0db-4075-9e72-31d61ffd0483","service":"FishService","method":"AnalyzeSwarm","input":{"equals":{"key":"stab2","greetings":{"hola":"mundo","merhaba":"dunya"},"cities":["Istanbul","Jakarta"]}},"output":{"data":{"hello":"world"}}}]`

				return httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.AddStub,
			expected: `["3f68f410-bb58-49ad-b679-23f2ed690c1d","6da11d72-c0db-4075-9e72-31d61ffd0483"]`,
		},
		{
			description: "add_error_stub_with_code",
			setupRequest: func() *http.Request {
				payload := `{"id":"cda7321b-9241-4a58-9cbf-0603e0146542","service":"InsectService","method":"TrackColony","input":{"contains":{"key":"value","greetings":{"hola":"mundo","merhaba":"dunya"},"cities":["Istanbul","Jakarta"]}},"output":{"error":"error msg","code":3}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.AddStub,
			expected: `["cda7321b-9241-4a58-9cbf-0603e0146542"]`,
		},
		{
			description: "trigger_error_stub_with_code",
			setupRequest: func() *http.Request {
				payload := `{"service":"InsectService","method":"TrackColony","data":{"key":"value","anotherKey":"anotherValue","greetings":{"hola":"mundo","merhaba":"dunya","hello":"world"},"cities":["Istanbul","Jakarta","Winterfell"]}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.SearchStubs,
			expected: `{"error":"error msg","code":3,"headers":null}` + "\n",
		},
		{
			description: "add_error_stub_without_code",
			setupRequest: func() *http.Request {
				payload := `{"id":"6d5ec9a6-94a7-4f23-b5ea-b04a37796adb","service":"ArachnidService","method":"StudyWeb","input":{"contains":{"key":"value","greetings":{"hola":"mundo","merhaba":"dunya"},"cities":["Istanbul","Jakarta"]}},"output":{"error":"error msg"}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.AddStub,
			expected: `["6d5ec9a6-94a7-4f23-b5ea-b04a37796adb"]`,
		},
		{
			description: "trigger_error_stub_without_code",
			setupRequest: func() *http.Request {
				payload := `{"service":"ArachnidService","method":"StudyWeb","data":{"key":"value","anotherKey":"anotherValue","greetings":{"hola":"mundo","merhaba":"dunya","hello":"world"},"cities":["Istanbul","Jakarta","Winterfell"]}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.SearchStubs,
			expected: `{"error":"error msg","headers":null}` + "\n",
		},
		{
			description: "match_nested_partial_stub",
			setupRequest: func() *http.Request {
				payload := `{"service":"BirdService","method":"TrackMigration","data":{"key":"value","anotherKey":"anotherValue","greetings":{"hola":"mundo","merhaba":"dunya","hello":"world"},"cities":["Istanbul","Jakarta","Winterfell"]}}`

				return httptest.NewRequest(http.MethodGet, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.SearchStubs,
			expected: `{"data":{"hello":"world"},"error":"","headers":null}` + "\n",
		},
		{
			description: "create_regex_stub",
			setupRequest: func() *http.Request {
				payload := `{"id":"faf39edb-c695-493f-a25e-ecfc171977dc","service":"MarineService","method":"AnalyzeCurrents","input":{"matches":{"field1":".*ello$"}},"output":{"data":{"reply":"OK"}}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.AddStub,
			expected: `["faf39edb-c695-493f-a25e-ecfc171977dc"]`,
		},
		{
			description: "match_regex_stub",
			setupRequest: func() *http.Request {
				payload := `{"service":"MarineService","method":"AnalyzeCurrents","data":{"field1":"hello"}}`

				return httptest.NewRequest(http.MethodGet, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.SearchStubs,
			expected: `{"data":{"reply":"OK"},"error":"","headers":null}` + "\n",
		},
		{
			description: "add_nested_regex_stub",
			setupRequest: func() *http.Request {
				payload := `{"id":"b1299ce3-a2a6-4fe7-94d4-0b68fc80afaa","service":"CrustaceanService","method":"TrackMolting","input":{"matches":{"key":"[a-z]{3}ue","greetings":{"hola":1,"merhaba":true,"hello":"^he[l]{2,}o$"},"cities":["Istanbul","Jakarta",".*"],"mixed":[5.5,false,".*"]}},"output":{"data":{"reply":"OK"}}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.AddStub,
			expected: `["b1299ce3-a2a6-4fe7-94d4-0b68fc80afaa"]`,
		},
		{
			description: "match_nested_regex_stub",
			setupRequest: func() *http.Request {
				payload := `{"service":"CrustaceanService","method":"TrackMolting","data":{"key":"value","greetings":{"hola":1,"merhaba":true,"hello":"helllllo"},"cities":["Istanbul","Jakarta","Gotham"],"mixed":[5.5,false,"Gotham"]}}`

				return httptest.NewRequest(http.MethodGet, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.SearchStubs,
			expected: `{"data":{"reply":"OK"},"error":"","headers":null}` + "\n",
		},
		{
			description: "fail_partial_match",
			setupRequest: func() *http.Request {
				payload := `{"service":"AmphibianService","method":"CheckMetamorphosis","data":{"field1":"hello field1"}}`

				return httptest.NewRequest(http.MethodGet, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.SearchStubs,
			expected: `{"error":"Can't find stub \n\nService: AmphibianService \n\nMethod: CheckMetamorphosis \n\nInput\n\n{\n\t\"field1\": \"hello field1\"\n}\n\nClosest Match \n\ncontains:{\n\t\"field1\": \"hello field1\",\n\t\"field3\": \"hello field3\"\n}"}`,
		},
		{
			description: "fail_exact_match",
			setupRequest: func() *http.Request {
				payload := `{"service":"MammalService","method":"CheckHabitat","data":{"Hola":"Dunia"}}`

				return httptest.NewRequest(http.MethodPost, "/api/stubs/search", bytes.NewReader([]byte(payload)))
			},
			endpoint: server.SearchStubs,
			expected: `{"error":"Can't find stub \n\nService: MammalService \n\nMethod: CheckHabitat \n\nInput\n\n{\n\t\"Hola\": \"Dunia\"\n}\n\nClosest Match \n\nequals:{\n\t\"Hola\": \"Mundo\"\n}"}`,
		},
	}

	for _, tc := range testScenarios {
		t.Run(tc.description, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := tc.setupRequest()
			tc.endpoint(recorder, request)
			result, err := io.ReadAll(recorder.Result().Body)
			require.NoError(t, err)
			require.JSONEq(t, tc.expected, string(result))
		})
	}

	t.Run("clear_all_stubs", func(t *testing.T) {
		deleteRecorder := httptest.NewRecorder()
		deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/stubs", nil)
		server.PurgeStubs(deleteRecorder, deleteRequest)

		listRecorder := httptest.NewRecorder()
		listRequest := httptest.NewRequest(http.MethodGet, "/api/stubs", nil)
		server.ListStubs(listRecorder, listRequest)
		result, _ := io.ReadAll(listRecorder.Result().Body)
		require.Equal(t, http.StatusNoContent, deleteRecorder.Result().StatusCode)
		require.JSONEq(t, "[]", string(result))
	})
}
