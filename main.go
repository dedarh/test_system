package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"flag"
	"io/ioutil"

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
	IdAnswer   int    `db:"i_answer"`
	Text       string `db:"text"`
	Correct    int    `db:"status"`
}

var db *sqlx.DB
var configFile = flag.String("config", "conf.json", "Where to read the config from")

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
	var id int
	var dbconnect = "postgresql://" + config.DbLogin + "@" + config.DbHost + ":26257/" + config.DbDb + "?sslmode=disable"

	db = sqlx.MustConnect("postgres", dbconnect)

	// 	db =	sqlx.MustConnect("postgres", "postgresql://root@localhost:26257/test_systems?sslmode=disable") //govnocode

	if err := db.Get(&id, "SELECT count(*) FROM users"); err != nil {
		log.Fatal(err)
	}
	log.Print("количество пользователей", id)
	//fmt.Printf("%#v\n%#v","количество пользователей", id)
}

func usersIndex(w http.ResponseWriter, r *http.Request) { //игформация о пользователе
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
		answerUser, err := json.Marshal(u) //govnocode
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Fprintf(w, string(answerUser))
	}
}

func usersShow(w http.ResponseWriter, r *http.Request) { //игформация об определенном пользователе
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

	var user []User

	if err := db.Select(&user, "SELECT * FROM users WHERE id_user  = $1", r.FormValue("id")); err != nil {
		http.NotFound(w, r)
		return
	}

	answerUser, err := json.Marshal(user[0]) //govnocode
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

	answerTest, err := json.Marshal(test) //govnocode
	if err != nil {
		log.Println(err)
		return
	}
	fmt.Fprintf(w, string(answerTest))
}

func testStart(w http.ResponseWriter, r *http.Request) { //создание теста и отправка его пользователю
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}

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

func testCheckQuestion(w http.ResponseWriter, r *http.Request) { //проверка верного ответа на вопрос  http://localhost:4080/test_check_qestion/?Id_Test=1&Answer_user=[{"Id_Question":1,"ID_Answer":1,"Text":"test%20text"},{"Id_Question":1,"ID_Answer":2,"Text":"test%20text"},{"Id_Question":1,"ID_Answer":3,"Text":"test%20text"},{"Id_Question":2,"ID_Answer":5,"Text":"%20"}]
	if r.Method != "GET" {
		http.Error(w, http.StatusText(405), 405)
		return
	}
	var countCorrect = 0
	var countWrong = 0
	var questionAnswerCorrect []TestQuestionAnswer

	var AnswerUserJson []TestQuestionAnswer
	userAnswer := []byte(r.FormValue("Answer_user"))

	err := json.Unmarshal(userAnswer, &AnswerUserJson)
	if err != nil {
		log.Println(err)
		return
	}
	for i := 0; i < len(AnswerUserJson); i++ {
		if err := db.Select(&questionAnswerCorrect, `SELECT i_question, i_answer, text, status FROM answer WHERE i_question = $1 AND i_answer = $2`, AnswerUserJson[i].IdQuestion, AnswerUserJson[i].IdAnswer); err != nil {
			http.Error(w, http.StatusText(500), 500)
			fmt.Println(err)
			return
		}
		if questionAnswerCorrect[i].Correct == 1 {
			//log.Println("верный овтет")
			countCorrect++
		} else {
			//log.Println("не верный овтет")
			countWrong++
		}
	}
	log.Println("Верных ответов: ", countCorrect)
	log.Println("Неверных ответов: ", countWrong)

	answerUsrTest, err := json.Marshal(AnswerUserJson)
	if err != nil {
		log.Println(err)
		return
	}

	fmt.Fprintf(w, string(answerUsrTest))
}

func main() {
	http.HandleFunc("/users/", usersIndex)
	http.HandleFunc("/users/show/", usersShow)
	http.HandleFunc("/get_test/", testIndex)
	http.HandleFunc("/testStart/", testStart)
	http.HandleFunc("/test_check_question/", testCheckQuestion)
	http.ListenAndServe(":4080", nil)
}
