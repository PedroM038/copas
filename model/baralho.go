package model

// Implementação do modelo de jogo copas

import (
	"fmt"
)

var Naipes = []string{"♠", "♥", "♦", "♣"}
var Valores = []string{"2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K", "A"}

type Carta struct {
	Valor string
	Naipe string
}

func (c Carta) String() string {
	return fmt.Sprintf("%s%s", c.Valor, c.Naipe)
}

func (c Carta) ValorNumerico() int {
	switch c.Valor {
	case "2":
		return 2
	case "3":
		return 3
	case "4":
		return 4
	case "5":
		return 5
	case "6":
		return 6
	case "7":
		return 7
	case "8":
		return 8
	case "9":
		return 9
	case "10":
		return 10
	case "J":
		return 11
	case "Q":
		return 12
	case "K":
		return 13
	case "A":
		return 14
	}
	return -1 // Valor inválido
}

func (c Carta) PontosCarta() int {
	if c.Naipe == "♥" {
		return 1
	}
	if c.Naipe == "♠" && c.Valor == "Q" {
		return 13 // Dama de Espadas vale 13 pontos
	}
	return 0
}

func criarBaralho() []Carta {
	var baralho []Carta
	for _, naipe := range Naipes {
		for _, valor := range Valores {
			baralho = append(baralho, Carta{Valor: valor, Naipe: naipe})
		}
	}
	return baralho
}
