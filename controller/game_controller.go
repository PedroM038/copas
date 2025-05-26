package controller

import (
    "copas/model"
    "copas/network"
    "copas/view"
    "encoding/json"
    "sync"
)

// GameController gerencia o fluxo do jogo, turnos e token.
type GameController struct {
    game   *model.Game
    netw   *network.Network
    player *model.Player
    mu     sync.Mutex
}

// NewGameController cria um novo controlador.
func NewGameController(game *model.Game, netw *network.Network, player *model.Player) *GameController {
    return &GameController{
        game:   game,
        netw:   netw,
        player: player,
    }
}

// Start inicia o loop principal do controlador.
func (gc *GameController) Start() {
    go gc.listenMessages()

    // O jogador 0 inicia o token na primeira rodada
    if gc.player.ID == 0 && gc.game.Rodada == 1 {
        tokenMsg, _ := network.NewTokenMessage(gc.player.ID, 1, gc.player.ID)
        gc.netw.Send(tokenMsg)
    }

    // Loop principal bloqueante para manter o processo vivo
    select {}
}

// listenMessages trata as mensagens recebidas pela rede.
func (gc *GameController) listenMessages() {
    for msg := range gc.netw.Receive() {
        switch msg.Type {
        case network.MessageToken:
            gc.handleToken(msg)
        case network.MessagePlay:
            gc.handlePlay(msg)
        case network.MessageState:
            gc.handleState(msg)
        }
    }
}

// ...existing code...

func (gc *GameController) handleToken(msg network.Message) {
    gc.mu.Lock()
    defer gc.mu.Unlock()

    // Só executa ação se for minha vez
    if gc.game.JogadorAtual().ID == gc.player.ID {
        view.ShowGameState(gc.game)
        view.ShowPlayerHand(gc.player)

        // Solicita jogada ao usuário
        var cartaEscolhida model.Card
        for {
            idx := view.PromptCardChoice(gc.player)
            if idx >= 0 && idx < len(gc.player.Mao) {
                carta := gc.player.Mao[idx]
                leadSuit := gc.game.GetLeadSuit()
                isFirstTrick := gc.game.Rodada == 1
                if gc.player.CanPlayCard(carta, leadSuit, isFirstTrick) {
                    cartaEscolhida = carta
                    break
                } else {
                    view.ShowMessage("Jogada inválida pelas regras. Escolha outra carta.")
                }
            }
        }

        // Atualiza estado local
        gc.game.JogarCarta(gc.player.ID, cartaEscolhida)

        // Avança para o próximo jogador
        gc.game.ProximoJogador()

        // Envia jogada para o anel
        payload, _ := json.Marshal(cartaEscolhida)
        playMsg := network.Message{
            Type:    network.MessagePlay,
            From:    gc.player.ID,
            To:      -1,
            Payload: payload,
        }
        gc.netw.Send(playMsg)

        // Sincroniza estado
        gc.broadcastState()
    }

    // Repassa o token para o próximo jogador
    nextID := gc.game.JogadorAtual().ID
    tokenMsg, _ := network.NewTokenMessage(gc.player.ID, nextID, nextID)
    gc.netw.Send(tokenMsg)
}

// handlePlay processa a jogada de outro jogador.
func (gc *GameController) handlePlay(msg network.Message) {
    var carta model.Card
    if err := json.Unmarshal(msg.Payload, &carta); err != nil {
        return
    }
    gc.mu.Lock()
    defer gc.mu.Unlock()
    gc.game.JogarCarta(msg.From, carta)

    // Se todos jogaram, calcula vencedor e inicia nova rodada
    if len(gc.game.Mesa) == len(gc.game.Jogadores) {
        gc.game.FinalizarRodada()
        gc.broadcastState()
    }
}
// handleState atualiza o estado do jogo recebido.
func (gc *GameController) handleState(msg network.Message) {
    var newGame model.Game
    if err := json.Unmarshal(msg.Payload, &newGame); err != nil {
        return
    }
    gc.mu.Lock()
    *gc.game = newGame
    gc.mu.Unlock()
    view.ShowGameState(gc.game)
}

// broadcastState envia o estado atualizado do jogo para todos.
func (gc *GameController) broadcastState() {
    payload, _ := json.Marshal(gc.game)
    stateMsg := network.Message{
        Type:    network.MessageState,
        From:    gc.player.ID,
        To:      -1,
        Payload: payload,
    }
    gc.netw.Send(stateMsg)
}