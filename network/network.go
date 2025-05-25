package network

import (
    "fmt"
    "net"
    "time"
)

// Network representa a configuração de rede de um jogador no anel.
type Network struct {
    LocalAddr  *net.UDPAddr // Endereço local deste jogador
    NextAddr   *net.UDPAddr // Endereço do próximo jogador no anel
    Conn       *net.UDPConn // Conexão UDP
    ReceiveCh  chan Message // Canal para mensagens recebidas
    SendCh     chan Message // Canal para mensagens a serem enviadas
    QuitCh     chan struct{}
}

// NewNetwork inicializa a rede para o jogador.
func NewNetwork(localAddr, nextAddr string) (*Network, error) {
    lAddr, err := net.ResolveUDPAddr("udp", localAddr)
    if err != nil {
        return nil, fmt.Errorf("erro ao resolver endereço local: %w", err)
    }
    nAddr, err := net.ResolveUDPAddr("udp", nextAddr)
    if err != nil {
        return nil, fmt.Errorf("erro ao resolver endereço do próximo: %w", err)
    }
    conn, err := net.ListenUDP("udp", lAddr)
    if err != nil {
        return nil, fmt.Errorf("erro ao abrir conexão UDP: %w", err)
    }
    netw := &Network{
        LocalAddr: lAddr,
        NextAddr:  nAddr,
        Conn:      conn,
        ReceiveCh: make(chan Message, 8),
        SendCh:    make(chan Message, 8),
        QuitCh:    make(chan struct{}),
    }
    go netw.listen()
    go netw.sender()
    return netw, nil
}

// listen escuta por mensagens UDP e as envia para ReceiveCh.
func (n *Network) listen() {
    buf := make([]byte, 4096)
    for {
        select {
        case <-n.QuitCh:
            return
        default:
            n.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
            nBytes, _, err := n.Conn.ReadFromUDP(buf)
            if err != nil {
                if ne, ok := err.(net.Error); ok && ne.Timeout() {
                    continue
                }
                continue
            }
            msg, err := DecodeMessage(buf[:nBytes])
            if err == nil {
                n.ReceiveCh <- msg
            }
        }
    }
}

// sender envia mensagens do canal SendCh para o próximo jogador.
func (n *Network) sender() {
    for {
        select {
        case <-n.QuitCh:
            return
        case msg := <-n.SendCh:
            data, err := EncodeMessage(msg)
            if err != nil {
                continue
            }
            n.Conn.WriteToUDP(data, n.NextAddr)
        }
    }
}

// Send envia uma mensagem para o próximo jogador.
func (n *Network) Send(msg Message) {
    n.SendCh <- msg
}

// Receive retorna o canal de mensagens recebidas.
func (n *Network) Receive() <-chan Message {
    return n.ReceiveCh
}

// Close encerra a conexão de rede.
func (n *Network) Close() {
    close(n.QuitCh)
    n.Conn.Close()
}