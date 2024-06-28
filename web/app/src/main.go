package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

type FrequenciaRequest struct {
	RA        string `json:"ra"`
	Bimestre  string `json:"bimestre"`
	DiaLetivo string `json:"diaLetivo"`
	Presente  bool   `json:"presente"`
}

type ComunicadoRequest struct {
	Mensagem string `json:"message"`
	Destino  string `json:"target"`
	Tipo     string `json:"type"`
}

type Bimestre struct {
	Frequencia map[string]bool
}

func (b *Bimestre) SetPresenca(diaLet string) {
	b.Frequencia[diaLet] = true
}

func (b *Bimestre) SetFalta(diaLet string) {
	b.Frequencia[diaLet] = false
}

type Aluno struct {
	RA        string
	Bimestres map[string]*Bimestre
}

func (a *Aluno) GetBimestre(bimestre string) *Bimestre {
	bim := a.Bimestres[bimestre]

	if bim == nil {
		a.Bimestres[bimestre] = &Bimestre{
			Frequencia: make(map[string]bool),
		}
		bim = a.Bimestres[bimestre]
	}

	return bim
}

func (a *Aluno) GetCurrentBimestre() *Bimestre {
	numBimestres := len(a.Bimestres)

	return a.GetBimestre(fmt.Sprintf("%d", numBimestres))
}

func (a *Aluno) IsFaltante() bool {
	bim := a.GetCurrentBimestre()

	contador := 0

	freqSlice := make([]bool, len(bim.Frequencia))
	for k, v := range bim.Frequencia {
		pos, _ := strconv.Atoi(k)
		freqSlice[pos-1] = v
	}

	for _, v := range freqSlice {
		if !v {
			contador++
			continue
		}

		contador = 0
	}

	return contador >= 3
}

type Classe map[string]*Aluno

func (c *Classe) GetAluno(ra string) *Aluno {
	aluno := (*c)[ra]

	if aluno == nil {
		(*c)[ra] = &Aluno{
			RA:        ra,
			Bimestres: make(map[string]*Bimestre),
		}
		aluno = (*c)[ra]
	}

	return aluno
}

func (c *Classe) SetFalta(ra, bimestre, diaLetivo string) {
	c.GetAluno(ra).GetBimestre(bimestre).SetFalta(diaLetivo)
}

func (c *Classe) SetPresenca(ra, bimestre, diaLetivo string) {
	c.GetAluno(ra).GetBimestre(bimestre).SetPresenca(diaLetivo)
}

func RequestValida(req *FrequenciaRequest) bool {
	return req.RA != "" && req.Bimestre != "" && req.DiaLetivo != ""
}

var classe = make(Classe)

func Main() {
	server := http.NewServeMux()

	server.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"status": "OK"}`)
	})

	boletim := http.NewServeMux()
	boletim.HandleFunc("/atualizar", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req FrequenciaRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if !RequestValida(&req) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		classe.SetFalta(req.RA, req.Bimestre, req.DiaLetivo)
		if req.Presente {
			classe.SetPresenca(req.RA, req.Bimestre, req.DiaLetivo)
		}

		if classe.GetAluno(req.RA).IsFaltante() {
			fmt.Printf("Aluno %s faltante, responsável foi comunicado.\n", req.RA)
		}

		io.WriteString(w, `{"message": "success"}`)
	})
	boletim.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"route": "/boletim"}`)
	})
	server.Handle("/boletim/", http.StripPrefix("/boletim", boletim))

	comunicados := http.NewServeMux()
	comunicados.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"route": "/comunicados"}`)
	})
	comunicados.HandleFunc("/enviar", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var req ComunicadoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if req.Mensagem == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if req.Tipo == "" {
			w.WriteHeader(http.StatusBadRequest)
		}

		tipos := []string{"classe", "responsavel", "geral"}
		if !slices.Contains(tipos, req.Tipo) {
			http.Error(w, fmt.Sprintf(
				`{"error": "tipo \"%s\" inválido, deve ser uma das seguintes: %s."}`,
				req.Tipo, strings.Join(tipos, ", "),
			), http.StatusBadRequest)
			return
		}

		if req.Tipo == "responsavel" {
			fmt.Printf("Mensagem enviada ao Responsável pelo aluno de RA %s: \"%s\"\n", req.Destino, req.Mensagem)
		}

		if req.Tipo == "classe" {
			fmt.Printf("Mensagem enviada a classe de nº %s: \"%s\"\n", req.Destino, req.Mensagem)
		}

		if req.Tipo == "geral" {
			fmt.Printf("Mensagem enviada a todos: \"%s\"\n", req.Mensagem)
		}
		io.WriteString(w, `{"message": "success"}`)
	})
	server.Handle("/comunicados/", http.StripPrefix("/comunicados", comunicados))

	http.ListenAndServe(":80", server)
}

func main() {
	Main()
}
