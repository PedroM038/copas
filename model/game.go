package model

import (
    "math/rand"
    "time"
)

// Game representa o estado do jogo Copas.
type Game struct {
    Jogadores []*Player `json:"jogadores"`
    Baralho   []Card    `json:"baralho"`
    Mesa      []Card    `json:"mesa"`      // Cartas jogadas na rodada atual
    Rodada    int       `json:"rodada"`    // Número da rodada atual
    AtivoID   int       `json:"ativo_id"`  // ID do jogador da vez
}

// NewGame cria um novo jogo com os jogadores fornecidos.
func NewGame(jogadores []*Player) *Game {
    baralho := NewDeck()
    rand.Seed(time.Now().UnixNano())
    rand.Shuffle(len(baralho), func(i, j int) { baralho[i], baralho[j] = baralho[j], baralho[i] })

    return &Game{
        Jogadores: jogadores,
        Baralho:   baralho,
        Mesa:      []Card{},
        Rodada:    1,
        AtivoID:   0,
    }
}

func (g *Game) DistribuirCartas() {
    numJogadores := len(g.Jogadores)
    numCartas := len(g.Baralho) / numJogadores
    for i, p := range g.Jogadores {
        p.Mao = []Card{} // Limpa a mão antes de distribuir
        p.Mao = append(p.Mao, g.Baralho[i*numCartas:(i+1)*numCartas]...)
    }
}

// ProximoJogador avança para o próximo jogador no anel.
func (g *Game) ProximoJogador() {
    g.AtivoID = (g.AtivoID + 1) % len(g.Jogadores)
}

// JogadorAtual retorna o jogador da vez.
func (g *Game) JogadorAtual() *Player {
    return g.Jogadores[g.AtivoID]
}

// JogarCarta adiciona uma carta à mesa e remove da mão do jogador.
func (g *Game) JogarCarta(jogadorID int, carta Card) bool {
    for _, p := range g.Jogadores {
        if p.ID == jogadorID {
            if p.RemoveCard(carta) {
                g.Mesa = append(g.Mesa, carta)
                return true
            }
        }
    }
    return false
}

// NovaRodada limpa a mesa e avança a rodada.
func (g *Game) NovaRodada() {
    g.Mesa = []Card{}
    g.Rodada++
}

// GetLeadSuit retorna o naipe da primeira carta jogada na rodada.
func (g *Game) GetLeadSuit() Naipe {
    if len(g.Mesa) > 0 {
        return g.Mesa[0].Naipe
    }
    return ""
}

// CalcularVencedorRodada determina quem ganhou a rodada.
func (g *Game) CalcularVencedorRodada() int {
    if len(g.Mesa) != len(g.Jogadores) {
        return -1 // Rodada não completa
    }
    
    leadSuit := g.Mesa[0].Naipe
    vencedorIdx := 0
    maiorValor := g.Mesa[0].Valor
    
    for i := 1; i < len(g.Mesa); i++ {
        carta := g.Mesa[i]
        // Só cartas do naipe principal podem vencer
        if carta.Naipe == leadSuit && carta.Valor > maiorValor {
            maiorValor = carta.Valor
            vencedorIdx = i
        }
    }
    
    return (g.AtivoID - len(g.Mesa) + 1 + vencedorIdx + len(g.Jogadores)) % len(g.Jogadores)
}

// ContarPontosRodada soma os pontos das cartas jogadas.
func (g *Game) ContarPontosRodada() int {
    pontos := 0
    for _, carta := range g.Mesa {
        pontos += carta.GetPoints()
    }
    return pontos
}

// FinalizarRodada atribui pontos ao vencedor e limpa a mesa.
func (g *Game) FinalizarRodada() {
    vencedorID := g.CalcularVencedorRodada()
    if vencedorID >= 0 {
        pontos := g.ContarPontosRodada()
        g.Jogadores[vencedorID].Pontos += pontos
        g.AtivoID = vencedorID // Vencedor inicia próxima rodada
    }
    g.NovaRodada()
}

func (g *Game) FimDaPartida() bool {
    for _, p := range g.Jogadores {
        if len(p.Mao) > 0 {
            return false
        }
    }
    return true
}