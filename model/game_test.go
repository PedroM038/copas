package model

import (
	"fmt"
	"testing"
)

// printHand imprime a mão de um jogador.
func printHand(player Player) {
	fmt.Printf("Mão de %s: ", player.Name)
	for _, card := range player.Hand {
		fmt.Printf("[%s] ", card.String())
	}
	fmt.Println()
}

// printScores imprime as pontuações dos jogadores.
func printScores(players []Player) {
	fmt.Println("Pontuações:")
	for _, p := range players {
		fmt.Printf("  %s: %d\n", p.Name, p.Score)
	}
}

func TestGameSimulation(t *testing.T) {
	playerNames := []string{"Alice", "Bob", "Carol", "Dave"}
	game, err := NewGame(playerNames)
	if err != nil {
		t.Fatalf("Erro ao criar jogo: %v", err)
	}

	rodada := 1
	for !game.GameOver {
		// Ordena as mãos para facilitar visualização
		for i := range game.Players {
			game.SortHand(i)
		}

		fmt.Printf("\n=== INÍCIO DA RODADA %d ===\n", rodada)
		for _, p := range game.Players {
			printHand(p)
		}
		fmt.Printf("Jogador inicial: %s\n", game.Players[game.CurrentPlayer].Name)

		for trickNum := 1; trickNum <= 13 && !game.GameOver; trickNum++ {
			fmt.Printf("\nVaza %d:\n", trickNum)
			for i := 0; i < 4 && !game.GameOver; i++ {
				player := &game.Players[game.CurrentPlayer]
				validPlays := game.GetValidPlays(player.ID)
				if len(validPlays) == 0 {
					t.Fatalf("Nenhuma jogada válida para %s", player.Name)
				}
				card := validPlays[0]
				err := game.PlayCard(player.ID, card)
				if err != nil {
					t.Fatalf("Jogada inválida de %s: %v", player.Name, err)
				}
				fmt.Printf("%s jogou %s\n", player.Name, card.String())
			}
			if len(game.CompletedTricks) > 0 {
				lastTrick := game.CompletedTricks[len(game.CompletedTricks)-1]
				winner := game.Players[lastTrick.WinnerID]
				fmt.Printf("Vaza ganha por %s\n", winner.Name)
			}
		}
		fmt.Println("\nPontuação parcial:")
		printScores(game.Players)
		rodada++
	}

	fmt.Println("\n=== FIM DE JOGO ===")
	printScores(game.Players)
	fmt.Printf("Vencedor: %s\n", game.Players[game.WinnerID].Name)
}
