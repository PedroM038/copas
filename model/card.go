package model

import "fmt"

// Naipe representa os naipes das cartas.
type Naipe string

const (
    NaipeCopas    Naipe = "Copas"
    NaipeEspadas  Naipe = "Espadas"
    NaipeOuros    Naipe = "Ouros"
    NaipePaus     Naipe = "Paus"
)

// Valor representa o valor de uma carta (2 a 14, onde 11=J, 12=Q, 13=K, 14=A).
type Valor int

const (
    Dois  Valor = 2
    Tres  Valor = 3
    Quatro Valor = 4
    Cinco Valor = 5
    Seis  Valor = 6
    Sete  Valor = 7
    Oito  Valor = 8
    Nove  Valor = 9
    Dez   Valor = 10
    Valete Valor = 11
    Dama   Valor = 12
    Rei    Valor = 13
    As     Valor = 14
)

// Card representa uma carta do baralho.
type Card struct {
    Naipe Naipe `json:"naipe"`
    Valor Valor `json:"valor"`
}

// String retorna a representação textual da carta.
func (c Card) String() string {
    valorStr := map[Valor]string{
        Valete: "J", Dama: "Q", Rei: "K", As: "A",
    }
    v, ok := valorStr[c.Valor]
    if !ok {
        v = fmt.Sprintf("%d", c.Valor)
    }
    return fmt.Sprintf("%s de %s", v, c.Naipe)
}

// NewDeck retorna um baralho tradicional de 52 cartas.
func NewDeck() []Card {
    naipes := []Naipe{NaipeCopas, NaipeEspadas, NaipeOuros, NaipePaus}
    valores := []Valor{Dois, Tres, Quatro, Cinco, Seis, Sete, Oito, Nove, Dez, Valete, Dama, Rei, As}
    deck := make([]Card, 0, 52)
    for _, n := range naipes {
        for _, v := range valores {
            deck = append(deck, Card{Naipe: n, Valor: v})
        }
    }
    return deck
}

// Sugestão para card.go
func (c Card) GetPoints() int {
    if c.Naipe == NaipeCopas {
        return 1  // Cada carta de Copas = 1 ponto
    }
    if c.Naipe == NaipeEspadas && c.Valor == Dama {
        return 13 // Dama de Espadas = 13 pontos
    }
    return 0 // Outras cartas = 0 pontos
}