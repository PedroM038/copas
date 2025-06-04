package model

type Jogador struct {
	ID           int
	Nome         string
	Cartas       []Carta
	Pontuacao    int
	CartasGanhas []Carta // utilizado para armazenar as cartas ganhas pelo jogador
}

func (j *Jogador) AdicionarCarta(carta Carta) {
	j.Cartas = append(j.Cartas, carta)
}

func (j *Jogador) RemoverCarta(carta Carta) {
	for i, c := range j.Cartas {
		if c == carta {
			j.Cartas = append(j.Cartas[:i], j.Cartas[i+1:]...)
			return
		}
	}
}

func (j *Jogador) CalcularPontuacao() int {
	pontos := 0
	for _, carta := range j.CartasGanhas {
		pontos += carta.PontosCarta()
	}
	return pontos
}

func (j *Jogador) TemCarta(carta Carta) bool {
	for _, c := range j.Cartas {
		if c == carta {
			return true
		}
	}
	return false
}

func (j *Jogador) TemNaipe(naipe string) bool {
	for _, carta := range j.Cartas {
		if carta.Naipe == naipe {
			return true
		}
	}
	return false
}
