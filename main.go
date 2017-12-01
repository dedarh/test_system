package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var config struct {
	DbLogin string `json:"dbLogin"`
	DbHost  string `json:"dbHost"`
	DbDb    string `json:"dbDb"`
}

type User struct {
	Id       int    `db:"id_user"`
	Login    string `db:"login"`
	Password string `db:"password"`
	Name     string `db:"name"`
	Gr       int    `db:"gr"`
	Token    string `db:"token"`
}

type Test struct {
	Id   int    `db:"i_test"`
	Name string `db:"name"`
}
type TestQuestion struct {
	IdQuestion string `db:"i_question"`
	Text       string `db:"question_name"`
	Type       string `db:"type"`
	Answer     []TestQuestionAnswer
}
type TestQuestionAnswer struct {
	IdQuestion int    `db:"i_question"`
	IDAnswer   int    `db:"i_answer"`
	Text       string `db:"text"`
}

var (
	db *sqlx.DB
	configFile = flag.String("config", "conf.json", "Where to read the config from")
)

func loadConfig() error {
	jsonData, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, &config)
}

func init() {
	if err := loadConfig(); err != nil {
		log.Fatal(err)
	}
	log.Println("Config loaded from", *configFile)

	db = sqlx.MustConnect("postgres", "postgresql://" + config.DbLogin + "@" + config.DbHost + ":26257/" + config.DbDb + "?sslmode=disable")

	var id int
	if err := db.Get(&id, "SELECT count(*) FROM users"); err != nil {
		log.Fatal(err)
	}
	log.Println("Users count:", id)
}

func usersIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	var users []User

	if err := db.Select(&users, "SELECT * FROM users ORDER BY Name ASC"); err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	for _, u := range users {
		answerUser, err := json.Marshal(u)
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Fprintf(w, string(answerUser))
	}
}

func usersShow(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	var user []User

	if err := db.Select(&user, "SELECT * FROM users WHERE id_user  = $1", r.FormValue("id")); err != nil {
		http.NotFound(w, r)
		return
	}

	answerUser, err := json.Marshal(user[0])
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintf(w, string(answerUser))
}

func testIndex(w http.ResponseWriter, r *http.Request) { //все тесты
	if r.Method != "POST" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	var test []Test

	if err := db.Select(&test, "SELECT * FROM test_name  ORDER BY name ASC"); err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	answerTest, err := json.Marshal(test)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintf(w, string(answerTest))
}

func testStart(w http.ResponseWriter, r *http.Request) { //все тесты шаблон
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	r.ParseForm()

	id := r.FormValue("id") //id теста

	var testQuestion []TestQuestion
	err := db.Select(&testQuestion, `SELECT q.question_name,t_q.i_question, q.type FROM "test_question" t_q JOIN "question" q ON t_q.i_question = q.i_question  WHERE t_q.i_test = $1 ORDER BY q.i_question DESC`, id)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		fmt.Println(err)
		return
	}

	for i := 1; i < len(testQuestion); i++ {
		var tqa []TestQuestionAnswer
		db.Select(&tqa, "SELECT i_question, i_answer, text FROM answer WHERE i_question = $1", testQuestion[i].IdQuestion)
		testQuestion[i].Answer = tqa
	}

	fmt.Println(testQuestion)
	answerTest, err := json.Marshal(testQuestion) //govnocode
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintf(w, string(answerTest))
}

func nextQuestion(w http.ResponseWriter, r *http.Request) { //все тесты шаблон
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	var questionAnswer []TestQuestionAnswer

	if err := db.Select(&questionAnswer, `SELECT i_question, i_answer, text FROM answer WHERE i_question = $1`, r.FormValue("Id_Question")); err != nil {
		http.Error(w, http.StatusText(500), 500)
		fmt.Println(err)
		return
	}

	fmt.Println(questionAnswer)
	answerTest, err := json.Marshal(questionAnswer)
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintf(w, string(answerTest))
}

func main() {
	http.HandleFunc("/users/", usersIndex)
	http.HandleFunc("/users/show/", usersShow)
	http.HandleFunc("/get_test/", testIndex)
	http.HandleFunc("/testStart/", testStart)
	http.HandleFunc("/next/", nextQuestion)
	http.ListenAndServe(":4080", nil)
}
