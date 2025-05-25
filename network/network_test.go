package network

import (
    "testing"
    "time"
)

func TestNetworkLoopback(t *testing.T) {
    // Usa a mesma porta para local e próximo (loopback)
    addr := "127.0.0.1:9000"
    netw, err := NewNetwork(addr, addr)
    if err != nil {
        t.Fatalf("Falha ao criar rede: %v", err)
    }
    defer netw.Close()

    // Cria uma mensagem simples
    msg := Message{
        Type: MessageToken,
        From: 1,
        To:   1,
        Payload: map[string]interface{}{
            "test": "ok",
        },
    }

    // Envia a mensagem
    netw.Send(msg)

    // Espera e verifica se a mensagem foi recebida
    select {
    case received := <-netw.Receive():
        if received.Type != msg.Type || received.From != msg.From || received.To != msg.To {
            t.Errorf("Mensagem recebida diferente da enviada: %+v", received)
        }
    case <-time.After(2 * time.Second):
        t.Error("Timeout esperando mensagem")
    }
}