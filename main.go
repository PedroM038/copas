package main

import (
    "copas/controller"
    "copas/model"
    "copas/network"
    "encoding/json"
    "fmt"
    "os"
    "strconv"
    "time"
)

// Payload da mensagem DISCOVERY
type DiscoveryPayload struct {
    IDs []int `json:"ids"`
}

func main() {
    if len(os.Args) != 4 {
        fmt.Println("Uso: go run main.go <player_id> <porta_local> <porta_proximo>")
        fmt.Println("Exemplo para 4 terminais:")
        fmt.Println("Terminal 1: go run main.go 0 9000 9001")
        fmt.Println("Terminal 2: go run main.go 1 9001 9002")
        fmt.Println("Terminal 3: go run main.go 2 9002 9003")
        fmt.Println("Terminal 4: go run main.go 3 9003 9000")
        return
    }

    playerID, err := strconv.Atoi(os.Args[1])
    if err != nil || playerID < 0 || playerID > 3 {
        fmt.Println("player_id deve ser 0, 1, 2 ou 3")
        return
    }
    localPort := os.Args[2]
    nextPort := os.Args[3]

    localAddr := "127.0.0.1:" + localPort
    nextAddr := "127.0.0.1:" + nextPort

    // Criar jogadores
    jogadores := []*model.Player{
        model.NewPlayer(0, "Alice"),
        model.NewPlayer(1, "Bob"),
        model.NewPlayer(2, "Carol"),
        model.NewPlayer(3, "Dave"),
    }

    game := model.NewGame(jogadores)
    player := jogadores[playerID]

    // Inicializar rede
    netw, err := network.NewNetwork(playerID, localAddr, nextAddr)
    if err != nil {
        fmt.Println("Erro ao inicializar rede:", err)
        return
    }
    defer netw.Close()

    fmt.Printf("Jogador %d (%s) iniciado na porta %s\n", playerID, player.Nome, localPort)
    fmt.Println("Aguardando outros jogadores se conectarem ao anel...")

    // Aguarda um tempo para todos os jogadores iniciarem suas redes
    time.Sleep(5 * time.Second)

    // Processo de descoberta do anel
    if playerID == 0 {
        fmt.Println("Iniciando descoberta do anel...")

        payload := DiscoveryPayload{IDs: []int{playerID}}
        payloadBytes, _ := json.Marshal(payload)
        discoveryMsg := network.Message{
            Type:    network.MessageType("DISCOVERY"),
            From:    playerID,
            To:      -1, // broadcast
            Payload: json.RawMessage(payloadBytes),
        }
        netw.Send(discoveryMsg)

        // Aguarda descoberta completa
        timeout := time.After(15 * time.Second)
        for {
            select {
            case msg := <-netw.Receive():
                if string(msg.Type) == "DISCOVERY" {
                    var p DiscoveryPayload
                    if err := json.Unmarshal(msg.Payload, &p); err != nil {
                        continue
                    }

                    // Se a mensagem voltou para mim e temos 4 jogadores
                    if msg.From == playerID && len(p.IDs) >= 4 {
                        fmt.Printf("Anel completo! Jogadores conectados: %v\n", p.IDs)
                        fmt.Println("Distribuindo cartas e iniciando o jogo...")

                        // Distribuir cartas
                        game.DistribuirCartas()

                        // Enviar mensagem de início
                        startMsg := network.Message{
                            Type:    network.MessageType("START"),
                            From:    playerID,
                            To:      -1,
                            Payload: nil,
                        }
                        netw.Send(startMsg)

                        goto gameStart
                    }

                    // Repassar mensagem para continuar circulação
                    netw.Send(msg)
                }
            case <-timeout:
                fmt.Println("Timeout: nem todos os jogadores estão conectados")
                fmt.Println("Certifique-se de que todos os 4 terminais estão rodando")
                return
            }
        }

    } else {
        // Jogadores 1, 2, 3 aguardam descoberta
        fmt.Println("Aguardando descoberta do anel...")

        for {
            select {
            case msg := <-netw.Receive():
                switch string(msg.Type) {
                case "DISCOVERY":
                    var p DiscoveryPayload
                    if err := json.Unmarshal(msg.Payload, &p); err != nil {
                        netw.Send(msg) // Repassa mesmo com erro
                        continue
                    }

                    // Adiciona meu ID se ainda não estiver na lista
                    alreadyIncluded := false
                    for _, id := range p.IDs {
                        if id == playerID {
                            alreadyIncluded = true
                            break
                        }
                    }

                    if !alreadyIncluded {
                        p.IDs = append(p.IDs, playerID)
                        fmt.Printf("Adicionado ao anel. Jogadores até agora: %v\n", p.IDs)
                    }

                    // Atualiza payload e repassa
                    newPayloadBytes, _ := json.Marshal(p)
                    newMsg := network.Message{
                        Type:    msg.Type,
                        From:    msg.From,
                        To:      -1,
                        Payload: json.RawMessage(newPayloadBytes),
                    }
                    netw.Send(newMsg)

                case "START":
                    fmt.Println("Anel completo! O jogo vai começar.")

                    // Distribuir cartas localmente
                    game.DistribuirCartas()

                    // Repassar mensagem se não foi originada por mim
                    if msg.From != playerID {
                        netw.Send(msg)
                    }
                    goto gameStart

                default:
                    // Repassar outras mensagens
                    netw.Send(msg)
                }
            case <-time.After(20 * time.Second):
                fmt.Println("Timeout: problema na descoberta do anel")
                return
            }
        }
    }

gameStart:
    fmt.Printf("\n🎮 JOGO DE COPAS INICIADO! 🎮\n")
    fmt.Printf("Jogador: %s (ID: %d)\n", player.Nome, player.ID)
    fmt.Printf("Cartas na mão: %d\n\n", len(player.Mao))

    // Pequena pausa para sincronização
    time.Sleep(2 * time.Second)

    // Iniciar controlador do jogo (ajustado para o novo construtor)
    ctrl := controller.NewGameController(game, netw, player)
    ctrl.Start()
}