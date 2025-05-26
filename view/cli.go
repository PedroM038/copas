package view

import (
    "fmt"
    "copas/model"
)

var lastGameState string

// Exibe informações básicas do jogo no terminal.
func ShowGameState(game *model.Game) {
    estado := fmt.Sprintf("Rodada:%d|Mesa:%d|Jogador:%d", 
        game.Rodada, len(game.Mesa), game.JogadorAtual().ID)
    
    // Só mostra se mudou
    if estado != lastGameState {
        fmt.Println("\n===== Estado do Jogo =====")
        fmt.Printf("Rodada: %d/13\n", game.Rodada)
        fmt.Printf("Jogador da vez: %s (ID %d)\n", game.JogadorAtual().Nome, game.JogadorAtual().ID)
        fmt.Printf("Cartas na mesa: %d/4\n", len(game.Mesa))
        
        if len(game.Mesa) > 0 {
            fmt.Println("Cartas jogadas:")
            for i, c := range game.Mesa {
                fmt.Printf("  [%d] %s\n", i+1, c.String())
            }
        }
        fmt.Println("==========================\n")
        lastGameState = estado
    }
}

// Exibe a mão do jogador.
func ShowPlayerHand(player *model.Player) {
    fmt.Printf("🎴 Mão de %s (%d cartas):\n", player.Nome, len(player.Mao))
    for i, c := range player.Mao {
        fmt.Printf("  [%d] %s\n", i+1, c.String())
    }
    fmt.Println()
}

// Solicita ao jogador que escolha uma carta para jogar.
func PromptCardChoice(player *model.Player) int {
    var escolha int
    fmt.Printf("Escolha o número da carta (1-%d): ", len(player.Mao))
    fmt.Scanln(&escolha)
    return escolha - 1 // índice começa em 0
}

// Exibe mensagem genérica.
func ShowMessage(msg string) {
    fmt.Println(msg)
}