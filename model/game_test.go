package model

import (
    "testing"
)

func TestGameModel(t *testing.T) {
    // Cria 4 jogadores
    jogadores := []*Player{
        NewPlayer(0, "Pedro"),
        NewPlayer(1, "Lio"),
        NewPlayer(2, "Eloiza"),
        NewPlayer(3, "Robertyo"),
    }

    // Cria o jogo
    game := NewGame(jogadores)

    // Distribui as cartas
    game.DistribuirCartas()
    for _, p := range game.Jogadores {
        if len(p.Mao) != 13 {
            t.Errorf("Jogador %s deveria ter 13 cartas, tem %d", p.Nome, len(p.Mao))
        }
    }

    // Jogador atual joga uma carta
    jogador := game.JogadorAtual()
    carta := jogador.Mao[0]
    ok := game.JogarCarta(jogador.ID, carta)
    if !ok {
        t.Errorf("Jogador %s não conseguiu jogar a carta", jogador.Nome)
    }
    if len(game.Mesa) != 1 {
        t.Errorf("Mesa deveria ter 1 carta, tem %d", len(game.Mesa))
    }
    if len(jogador.Mao) != 12 {
        t.Errorf("Jogador deveria ter 12 cartas após jogar, tem %d", len(jogador.Mao))
    }

    // Avança para o próximo jogador
    game.ProximoJogador()
    if game.JogadorAtual().ID == jogador.ID {
        t.Errorf("Não avançou para o próximo jogador")
    }

    // Nova rodada limpa a mesa e incrementa a rodada
    game.NovaRodada()
    if len(game.Mesa) != 0 {
        t.Errorf("Mesa deveria estar vazia após nova rodada")
    }
    if game.Rodada != 2 {
        t.Errorf("Rodada deveria ser 2, é %d", game.Rodada)
    }
}