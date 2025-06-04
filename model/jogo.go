package model

import (
	"fmt"
	"math/rand"
	"strings"
)

type Jogo struct {
	Jogadores      [4]*Jogador
	Baralho        []Carta
	TruqueAtual    *Truque
	Truques        []Truque
	RodadaAtual    int
	JogadorInicial int
	CopasQuebradas bool
	PrimeiraRodada bool
}

func NovoJogo() *Jogo {
	jogo := &Jogo{
		Baralho:        criarBaralho(),
		RodadaAtual:    1,
		CopasQuebradas: false,
		PrimeiraRodada: true,
	}

	// Criar jogadores
	for i := range 4 {
		jogo.Jogadores[i] = &Jogador{
			ID:           i,
			Nome:         fmt.Sprintf("Jogador %d", i+1),
			Cartas:       make([]Carta, 0),
			CartasGanhas: make([]Carta, 0),
		}
	}
	return jogo
}

func (j *Jogo) EmbaralharCartas() {
	rand.Shuffle(len(j.Baralho), func(i, k int) {
		j.Baralho[i], j.Baralho[k] = j.Baralho[k], j.Baralho[i]
	})
}

func (j *Jogo) DistribuirCartas() {
	j.EmbaralharCartas()

	// Distribuir 13 cartas para cada jogador
	for i := 0; i < 52; i++ {
		jogadorIdx := i % 4
		j.Jogadores[jogadorIdx].AdicionarCarta(j.Baralho[i])
	}

	// Encontrar quem tem o 2 de paus
	for i, jogador := range j.Jogadores {
		if jogador.TemCarta(Carta{Valor: "2", Naipe: "♣"}) {
			j.JogadorInicial = i
			break
		}
	}
}

func (j *Jogo) IniciarTruque(jogadorInicial int) {
	j.TruqueAtual = &Truque{
		CartasJogadas: make([]CartaJogada, 0),
		Vencedor:      nil,
	}
}

func (j *Jogo) JogarCarta(jogador *Jogador, carta Carta) error {
	if !jogador.TemCarta(carta) {
		return fmt.Errorf("jogador %s não tem a carta %s", jogador.Nome, carta)
	}

	if err := j.validarJogada(jogador, carta); err != nil {
		return err
	}

	jogador.RemoverCarta(carta)

	cartaJogada := CartaJogada{
		Carta:   carta,
		Jogador: jogador,
	}

	j.TruqueAtual.CartasJogadas = append(j.TruqueAtual.CartasJogadas, cartaJogada)

	if len(j.TruqueAtual.CartasJogadas) == 1 {
		j.TruqueAtual.NaipeTruco = carta.Naipe
	}

	if carta.Naipe == "♥" {
		j.CopasQuebradas = true
	}

	return nil
}

func (j *Jogo) validarJogada(jogador *Jogador, carta Carta) error {
	// Se é a primeira carta do primeiro truque, deve ser 2 de paus
	if j.PrimeiraRodada && len(j.TruqueAtual.CartasJogadas) == 0 {
		if carta.Valor != "2" || carta.Naipe != "♣" {
			return fmt.Errorf("primeira jogada deve ser 2 de paus")
		}
		return nil
	}

	// Se é o primeiro truque, não pode jogar copas nem dama de espadas
	if j.PrimeiraRodada {
		if carta.Naipe == "♥" || (carta.Valor == "Q" && carta.Naipe == "♠") {
			// Só pode jogar se não tiver outra opção
			temOutraOpcao := false
			for _, c := range jogador.Cartas {
				if c.Naipe != "♥" && !(c.Valor == "Q" && c.Naipe == "♠") {
					if len(j.TruqueAtual.CartasJogadas) == 0 || c.Naipe == j.TruqueAtual.NaipeTruco {
						temOutraOpcao = true
						break
					}
				}
			}
			if temOutraOpcao {
				return fmt.Errorf("não pode jogar copas ou dama de espadas no primeiro truque")
			}
		}
	}
	// Se não é a primeira carta do truque
	if len(j.TruqueAtual.CartasJogadas) > 0 {
		naipeInicial := j.TruqueAtual.NaipeTruco

		// Deve seguir o naipe se tiver
		if carta.Naipe != naipeInicial && jogador.TemNaipe(naipeInicial) {
			return fmt.Errorf("deve seguir o naipe %s", naipeInicial)
		}
	} else {
		// Se é a primeira carta e copas não foram quebradas
		if carta.Naipe == "♥" && !j.CopasQuebradas {
			// Só pode jogar copas se só tiver copas
			temOutroNaipe := false
			for _, c := range jogador.Cartas {
				if c.Naipe != "♥" {
					temOutroNaipe = true
					break
				}
			}
			if temOutroNaipe {
				return fmt.Errorf("copas ainda não foram quebradas")
			}
		}
	}

	return nil
}

func (j *Jogo) FinalizarTruque() {
	if len(j.TruqueAtual.CartasJogadas) != 4 {
		return
	}

	// Encontrar o vencedor do truque
	vencedor := j.encontrarVencedorTruque()
	j.TruqueAtual.Vencedor = vencedor

	// Dar todas as cartas para o vencedor
	for _, cartaJogada := range j.TruqueAtual.CartasJogadas {
		vencedor.CartasGanhas = append(vencedor.CartasGanhas, cartaJogada.Carta)
	}

	// Adicionar truque ao histórico
	j.Truques = append(j.Truques, *j.TruqueAtual)

	// Próximo truque será iniciado pelo vencedor
	j.JogadorInicial = vencedor.ID

	// Marcar que não é mais a primeira rodada
	if j.PrimeiraRodada {
		j.PrimeiraRodada = false
	}

	j.TruqueAtual = nil
}

func (j *Jogo) encontrarVencedorTruque() *Jogador {
	naipeInicial := j.TruqueAtual.NaipeTruco
	maiorValor := -1
	var vencedor *Jogador

	for _, cartaJogada := range j.TruqueAtual.CartasJogadas {
		if cartaJogada.Carta.Naipe == naipeInicial {
			valor := cartaJogada.Carta.ValorNumerico()
			if valor > maiorValor {
				maiorValor = valor
				vencedor = cartaJogada.Jogador
			}
		}
	}

	return vencedor
}

func (j *Jogo) RodadaCompleta() bool {
	return len(j.Truques) == 13
}

func (j *Jogo) CalcularPontuacaoRodada() {
	for _, jogador := range j.Jogadores {
		pontos := jogador.CalcularPontuacao()

		// Verificar shoot the moonS
		if pontos == 26 {
			// Jogador fez shoot the moon - todos os outros ganham 26 pontos
			for i, outroJogador := range j.Jogadores {
				if i != jogador.ID {
					outroJogador.Pontuacao += 26
				}
			}
		} else {
			jogador.Pontuacao += pontos
		}
	}
}

func (j *Jogo) JogoTerminado() bool {
	for _, jogador := range j.Jogadores {
		if jogador.Pontuacao >= 100 {
			return true
		}
	}
	return false
}

func (j *Jogo) Vencedor() *Jogador {
	if !j.JogoTerminado() {
		return nil
	}

	menorPontuacao := j.Jogadores[0].Pontuacao
	vencedor := j.Jogadores[0]

	for _, jogador := range j.Jogadores[1:] {
		if jogador.Pontuacao < menorPontuacao {
			menorPontuacao = jogador.Pontuacao
			vencedor = jogador
		}
	}

	return vencedor
}

func (j *Jogo) NovaRodada() {
	// Limpar dados da rodada anterior
	for _, jogador := range j.Jogadores {
		jogador.Cartas = make([]Carta, 0)
		jogador.CartasGanhas = make([]Carta, 0)
	}

	j.Truques = make([]Truque, 0)
	j.TruqueAtual = nil
	j.CopasQuebradas = false
	j.PrimeiraRodada = true
	j.RodadaAtual++

	// Redistribuir cartas
	j.DistribuirCartas()
}

func (j *Jogo) EstadoJogo() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("=== RODADA %d ===\n", j.RodadaAtual))
	sb.WriteString(fmt.Sprintf("Copas quebradas: %t\n", j.CopasQuebradas))
	sb.WriteString(fmt.Sprintf("Próximo jogador: %s\n", j.Jogadores[j.JogadorInicial].Nome))
	sb.WriteString("\nPontuações:\n")

	for _, jogador := range j.Jogadores {
		sb.WriteString(fmt.Sprintf("%s: %d pontos\n", jogador.Nome, jogador.Pontuacao))
	}

	if j.TruqueAtual != nil && len(j.TruqueAtual.CartasJogadas) > 0 {
		sb.WriteString("\nTruque atual:\n")
		for _, cartaJogada := range j.TruqueAtual.CartasJogadas {
			sb.WriteString(fmt.Sprintf("%s jogou %s\n",
				cartaJogada.Jogador.Nome, cartaJogada.Carta))
		}
	}

	return sb.String()
}
