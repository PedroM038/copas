package model

// Player representa um jogador do jogo Copas.
type Player struct {
    ID      int    `json:"id"`
    Nome    string `json:"nome"`
    Mao     []Card `json:"mao"`     // Cartas na mão do jogador
    Pontos  int    `json:"pontos"`  // Pontuação acumulada
    Ativo   bool   `json:"ativo"`   // Indica se é a vez do jogador
}

// NewPlayer cria um novo jogador.
func NewPlayer(id int, nome string) *Player {
    return &Player{
        ID:     id,
        Nome:   nome,
        Mao:    []Card{},
        Pontos: 0,
        Ativo:  false,
    }
}

// AddCard adiciona uma carta à mão do jogador.
func (p *Player) AddCard(c Card) {
    p.Mao = append(p.Mao, c)
}

// RemoveCard remove uma carta da mão do jogador (se existir).
func (p *Player) RemoveCard(c Card) bool {
    for i, card := range p.Mao {
        if card == c {
            p.Mao = append(p.Mao[:i], p.Mao[i+1:]...)
            return true
        }
    }
    return false
}

// Reset limpa a mão do jogador e zera o status de ativo.
func (p *Player) Reset() {
    p.Mao = []Card{}
    p.Ativo = false
}

// CanPlayCard verifica se o jogador pode jogar a carta (regras de Copas).
func (p *Player) CanPlayCard(c Card, leadSuit Naipe, isFirstTrick bool) bool {
    // Primeira rodada: não pode jogar Copas ou Dama de Espadas
    if isFirstTrick {
        if c.Naipe == NaipeCopas || (c.Naipe == NaipeEspadas && c.Valor == Dama) {
            return false
        }
    }
    
    // Se há um naipe sendo seguido, deve seguir se possível
    if leadSuit != "" && p.HasSuit(leadSuit) {
        return c.Naipe == leadSuit
    }
    
    return true // Pode jogar qualquer carta se não tem o naipe
}

// HasSuit verifica se o jogador tem cartas do naipe especificado.
func (p *Player) HasSuit(naipe Naipe) bool {
    for _, card := range p.Mao {
        if card.Naipe == naipe {
            return true
        }
    }
    return false
}