package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var acces bool = true
var verification bool = false
var tentative int
var motChoisi string
var motAffiche []string
var lettresDevinees []string
var essaisRestants int
var motifPenduASCII = []string{
	``,
	`
=========`,
	`
       |
       |
       |
       |
       |
=========`,
	`
   +---+
       |
       |
       |
       |
       |
=========`,
	`
  +---+
  |   |
      |
      |
      |
      |
=========`,
	`
  +---+
  |   |
  O   |
      |
      |
      |
=========`,
	`
  +---+
  |   |
  O   |
  |   |
      |
      |
=========`,
	`
  +---+
  |   |
  O   |
 /|   |
      |
      |
=========`,
	`
  +---+
  |   |
  O   |
 /|\  |
      |
      |
=========`,
	`
  +---+
  |   |
  O   |
 /|\  |
 /    |
      |
=========`,
	`
  +---+
  |   |
  O   |
 /|\  |
 / \  |
      |
=========`,
}

type Livre struct {
	Mot string
}

type Identifiant struct {
	Pseudo     string
	MotDePasse string
}

var identifiants []Identifiant

func recupererIdentifiants() ([]Identifiant, error) {
	db, err := sql.Open("sqlite3", "mots.db")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	// Préparer la requête SQL pour sélectionner tous les identifiants de la table "Identifiant"
	rows, err := db.Query("SELECT pseudo, mdp FROM Identifiant")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var identifiants []Identifiant
	for rows.Next() {
		var pseudo, mdp string
		err := rows.Scan(&pseudo, &mdp)
		if err != nil {
			return nil, err
		}
		identifiants = append(identifiants, Identifiant{Pseudo: pseudo, MotDePasse: mdp})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return identifiants, nil
}

func choisirMotBase() string {
	rand.Seed(time.Now().UnixNano())
	db, err := sql.Open("sqlite3", "mots.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query("SELECT mot FROM Mots")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var mots []string
	for rows.Next() {
		var mot string
		err := rows.Scan(&mot)
		if err != nil {
			log.Fatal(err)
		}
		mots = append(mots, mot)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	randomIndex := rand.Intn(len(mots))
	return mots[randomIndex]
}

func afficherMot() string {
	affichage := ""
	for _, char := range motChoisi {
		if contientLettre(lettresDevinees, string(char)) {
			affichage += string(char) + " "
		} else {
			affichage += "_ "
		}
	}
	return strings.TrimSpace(affichage)
}

func contientLettre(slice []string, element string) bool {
	for _, i := range slice {
		if i == element {
			return true
		}
	}
	return false
}

func resetGame() {
	motChoisi = choisirMotBase()
	lettresDevinees = []string{}
	essaisRestants = len(motifPenduASCII) - 1
	reveler := make([]bool, len(motChoisi))
	numAreveler := len(motChoisi)/2 - 1
	if numAreveler < 0 {
		numAreveler = 0
	}
	for i := 0; i < numAreveler; i++ {
		randomPosition := rand.Intn(len(motChoisi))
		if !reveler[randomPosition] {
			reveler[randomPosition] = true
		} else {
			i--
		}
	}
	motAffiche = make([]string, len(motChoisi))
	for i, char := range motChoisi {
		if reveler[i] {
			motAffiche[i] = string(char)
		} else {
			motAffiche[i] = "_"
		}
	}
}

func jouerHandler(w http.ResponseWriter, r *http.Request) {
	if verification {
		tmpl, err := template.ParseFiles("statics/home.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	if verification {
		if r.Method == http.MethodPost {
			r.ParseForm()
			devinette := strings.ToLower(r.Form.Get("lettre"))

			if len(devinette) == 1 && 'a' <= devinette[0] && devinette[0] <= 'z' {
				if !contientLettre(lettresDevinees, devinette) {
					lettresDevinees = append(lettresDevinees, devinette)

					if !strings.Contains(motChoisi, devinette) {
						essaisRestants--
					}
				}
			}
		}
		data := struct {
			Mot     string
			Lettres string
			Essais  int
			Pendu   string
		}{
			Mot:     afficherMot(),
			Lettres: strings.Join(lettresDevinees, ", "),
			Essais:  essaisRestants,
			Pendu:   motifPenduASCII[len(motifPenduASCII)-essaisRestants-1],
		}
		tmpl, err := template.ParseFiles("statics/hangman.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	resetGame()
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func main() {
	resetGame()
	var err error
	identifiants, err = recupererIdentifiants() // Récupérer les identifiants depuis la base de données
	if err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("/home", jouerHandler)
	http.HandleFunc("/hangman", handler)
	http.HandleFunc("/reset", resetHandler)
	http.HandleFunc("/", loginPage)
	http.HandleFunc("/authenticate", login)
	fs := http.FileServer(http.Dir("statics"))
	http.Handle("/statics/", http.StripPrefix("/statics/", fs))
	port := 8081
	fmt.Printf("Serveur en cours d'exécution sur le port %d...\n", port)
	err = http.ListenAndServe(fmt.Sprintf("localhost:%d", port), nil)
	if err != nil {
		fmt.Println(err)
	}
}

func loginPage(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("statics/login.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	if !acces {
		io.WriteString(w, "Vous n'avez plus de tentative")
		return
	}
	if r.Method == http.MethodPost {
		username := r.FormValue("username")
		password := r.FormValue("password")
		if username == "" || password == "" {
			http.Error(w, "Veuillez remplir tous les champs", http.StatusBadRequest)
			return
		}
		if containsForbiddenChars(username) || containsForbiddenChars(password) {
			http.Error(w, "Caractères non autorisés dans les identifiants", http.StatusBadRequest)
			return
		}
		pass := Hash("livre" + password)
		for _, id := range identifiants {
			if username == id.Pseudo && pass == id.MotDePasse {
				verification = true
				http.Redirect(w, r, "/home", http.StatusSeeOther)
				return
			}
		}
	}
	tentative++
	if tentative == 5 {
		io.WriteString(w, "Vous n'avez plus de tentative")
		acces = false
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func Hash(password string) string {
	hash := sha256.New()
	hash.Write([]byte(password))
	hashInBytes := hash.Sum(nil)
	return hex.EncodeToString(hashInBytes)
}

func containsForbiddenChars(input string) bool {
	for _, char := range []string{"'", `\`, "#", ";", ")", ","} {
		if strings.Contains(input, char) {
			return true
		}
	}
	return false
}

// func extraire() ([]string, error) {
// 	content, erreur := io.ReadFile("sel.txt")
// 	if erreur != nil {
// 		//si il y'a une erreur
// 		return nil, erreur
// 	}
// 	mots := strings.Fields(string(content))
// 	return mots, nil
// }

// func choisirMotSel() string {
// 	rand.Seed(time.Now().UnixNano())
// 	mot, erreur := "extraire()"
// 	if erreur != nil {
// 		log.Fatal(erreur)
// 	}
// 	return mot[rand.Intn(len(mot))]
// }
