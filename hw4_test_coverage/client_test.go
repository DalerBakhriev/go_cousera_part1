package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	// "bytes"
	"encoding/json"
	"encoding/xml"
	"time"
	// "io"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"strings"
)

const (
	rightToken                  string = "right_token"
	InternalErrorQuery          string = "internalError"
	InvalidToken                string = "invalid_token"
	BadRequestErrorQuery        string = "bad_request_query"
	BadRequestUnknownErrorQuery string = "bad_request_unknown_query"
	TimeoutErrorQuery           string = "timeout_query"
	InvalidJsonErrorQuery       string = "invalid_json_query"
)

type UserForParsing struct {
	Id        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

type UsersForParcing struct {
	UsersLst []UserForParsing `xml:"row"`
}

type TestCase struct {
	TestRequest       SearchRequest
	TestResponse      SearchResponse
	TestErrorResponse SearchErrorResponse
	AccessToken       string
	URL               string
	ErrorContains     string
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	file, err := os.Open("dataset.xml")
	if err != nil {
		fmt.Println("Can't open file", err)
	}
	defer file.Close()

	// Парсим xml в специально подготовленную структуту
	fileBytes, _ := ioutil.ReadAll(file)
	parsedUsers := UsersForParcing{}
	xml.Unmarshal(fileBytes, &parsedUsers)

	// Делаем список из пользователей
	users := make([]User, 0)
	for _, singleUserParsed := range parsedUsers.UsersLst {
		user := User{
			Id:     singleUserParsed.Id,
			Name:   singleUserParsed.FirstName + " " + singleUserParsed.LastName,
			Age:    singleUserParsed.Age,
			About:  singleUserParsed.About,
			Gender: singleUserParsed.Gender,
		}
		users = append(users, user)
	}

	// Проверка токена на совпадение
	searchToken := r.Header.Get("AccessToken")
	if searchToken == InvalidToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Поле, по которому сортируем.
	orderField := r.FormValue("order_field")

	if orderField == "" {
		orderField = "Name"
	}

	// Проверяем, есть ли в поле order_field одно из установленных полей
	knownOrderField := false
	for _, fieldName := range []string{"Id", "Age", "Name"} {
		if orderField == fieldName {
			knownOrderField = true
		}
	}

	if !knownOrderField {
		response, err := json.Marshal(SearchErrorResponse{"ErrorBadOrderField"})
		if err != nil {
			fmt.Println("Can't pack an error message to json", err)
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write(response)
		return
	}

	// подстрока в 1 из полей
	query := r.FormValue("query")

	if query == TimeoutErrorQuery {
		time.Sleep(time.Second * 2)
	}

	if query == BadRequestUnknownErrorQuery {
		resp, _ := json.Marshal(SearchErrorResponse{"UnknownError"})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(resp)
		return
	}

	if query == BadRequestErrorQuery {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if query == InvalidJsonErrorQuery {
		w.Write([]byte("invalid_json"))
		return
	}

	if query == InternalErrorQuery {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	usersFiltered := make([]User, 0)
	if query != "" {
		for _, singleUser := range users {
			if strings.Contains(singleUser.Name, query) || strings.Contains(singleUser.About, query) {
				usersFiltered = append(usersFiltered, singleUser)
			}
		}
	} else {
		usersFiltered = users
	}

	// Как сортируем. -1 по убыванию, 0 как встретилось, 1 по возрастанию
	orderBy, err := strconv.Atoi(r.FormValue("order_by"))
	if err != nil {
		fmt.Println("Can't decode order_by into int", err)
		return
	}

	// Сортировка по полю и по возрастанию/убыванию
	if orderField == "Id" && orderBy == -1 {
		sort.Slice(usersFiltered, func(i, j int) bool {
			return usersFiltered[i].Id > usersFiltered[j].Id
		})
	} else if orderField == "Id" && orderBy == 1 {
		sort.Slice(usersFiltered, func(i, j int) bool {
			return usersFiltered[i].Id < usersFiltered[j].Id
		})
	} else if orderField == "Age" && orderBy == -1 {
		sort.Slice(usersFiltered, func(i, j int) bool {
			return usersFiltered[i].Age > usersFiltered[j].Age
		})
	} else if orderField == "Age" && orderBy == 1 {
		sort.Slice(usersFiltered, func(i, j int) bool {
			return usersFiltered[i].Age < usersFiltered[j].Age
		})
	} else if orderField == "Name" && orderBy == -1 {
		sort.Slice(usersFiltered, func(i, j int) bool {
			return usersFiltered[i].Name > usersFiltered[j].Name
		})
	} else if orderField == "Name" && orderBy == 1 {
		sort.Slice(usersFiltered, func(i, j int) bool {
			return usersFiltered[i].Name < usersFiltered[j].Name
		})
	}

	offset, err := strconv.Atoi(r.FormValue("offset"))
	if err != nil {
		fmt.Println("Can't convert offset into int", err)
		return
	}

	if offset < 0 {
		response, err := json.Marshal(SearchErrorResponse{"Offset must be > 0"})
		if err != nil {
			fmt.Println("Can't pack an error message to json", err)
		}
		w.Write(response)
		return
	}

	limit, err := strconv.Atoi(r.FormValue("limit"))
	if err != nil {
		fmt.Println("Can't convert offset into int", err)
		return
	}

	if limit < 0 {
		response, err := json.Marshal(SearchErrorResponse{"Limit must be > 0"})
		if err != nil {
			fmt.Println("Can't pack an error message to json", err)
		}
		w.Write(response)
		return
	}

	response, err := json.Marshal(usersFiltered[offset:limit])
	if err != nil {
		fmt.Println("Can't convert response to json", err)
		return
	}
	w.Write(response)
}

func TestFindUsersErrors(t *testing.T) {
	cases := []TestCase{
		TestCase{
			TestRequest: SearchRequest{
				Limit: -1,
			},
			TestErrorResponse: SearchErrorResponse{
				Error: "limit must be > 0",
			},
		},
		TestCase{
			TestRequest: SearchRequest{
				Offset: -1,
			},
			TestErrorResponse: SearchErrorResponse{
				Error: "offset must be > 0",
			},
		},
		TestCase{
			TestRequest: SearchRequest{
				Query: InternalErrorQuery,
			},
			TestErrorResponse: SearchErrorResponse{
				Error: "SearchServer fatal error",
			},
		},
		TestCase{
			TestRequest: SearchRequest{
				OrderField: "SomeField",
				Limit:      26,
			},
			TestErrorResponse: SearchErrorResponse{
				Error: "OrderFeld SomeField invalid",
			},
		},
		TestCase{
			AccessToken: InvalidToken,
			TestErrorResponse: SearchErrorResponse{
				Error: "Bad AccessToken",
			},
		},
		TestCase{
			TestRequest: SearchRequest{
				Query: BadRequestUnknownErrorQuery,
			},
			ErrorContains: "unknown bad request error",
		},
		TestCase{
			TestRequest: SearchRequest{
				Query: TimeoutErrorQuery,
			},
			ErrorContains: "timeout for",
		},
		TestCase{
			TestRequest: SearchRequest{
				Query: InvalidJsonErrorQuery,
			},
			ErrorContains: "cant unpack result json",
		},
		TestCase{
			TestRequest: SearchRequest{
				Query: BadRequestErrorQuery,
			},
			ErrorContains: "cant unpack error json",
		},
		TestCase{
			URL: "someURL",
			ErrorContains: "unknown error",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(SearchServer))
	for caseNum, item := range cases {
		url := server.URL
		if item.URL != "" {
			url = item.URL
		}
		client := SearchClient{
			AccessToken: item.AccessToken,
			URL:         url,
		}

		response, err := client.FindUsers(item.TestRequest)

		if response != nil || err == nil {
			t.Errorf("[%d] expected error, got nil", caseNum)
		}

		if item.TestErrorResponse.Error  != "" && err.Error() != item.TestErrorResponse.Error  {
			t.Errorf("[%d] wrong result, expected %#v, got %#v", caseNum, item.TestErrorResponse.Error, err.Error())
		}

		if item.ErrorContains != "" && !strings.Contains(err.Error(), item.ErrorContains) {
			t.Errorf("[%d] wrong result, expected %#v to contain %#v", caseNum, err.Error(), item.ErrorContains)
		}
	}
}

func TestFindUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer server.Close()

	cases := []TestCase{
		TestCase{
			TestRequest: SearchRequest{Limit: 1},
		},
		TestCase{
			TestRequest: SearchRequest{Limit: 30},
		},
		TestCase{
			TestRequest: SearchRequest{Limit: 25, Offset: 1},
		},
	}

	for caseNum, item := range cases {
		client := SearchClient{
			URL: server.URL,
		}
		response, err := client.FindUsers(item.TestRequest)

		// we just need to cover 100% - so need in real testing
		if response == nil || err != nil {
			t.Errorf("[%d] expected response, got error", caseNum)
		}
	}
}
