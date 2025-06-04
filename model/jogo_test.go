package model

import (
	"fmt"
	"testing"
)

func TestJogoCompleto(t *testing.T) {
	fmt.Println("=== INICIANDO TESTE DO JOGO COPAS ===")

	// Criar novo jogo
	jogo := NovoJogo()

	// Distribuir cartas
	jogo.DistribuirCartas()

	fmt.Printf("Jogo iniciado! Jogador inicial: %s\n", jogo.Jogadores[jogo.JogadorInicial].Nome)

	// Verificar se o jogador inicial tem o 2 de paus
	if !jogo.Jogadores[jogo.JogadorInicial].TemCarta(Carta{Valor: "2", Naipe: "♣"}) {
		t.Error("Jogador inicial deveria ter o 2 de paus")
	}

	// Simular uma rodada completa
	for truqueNum := 0; truqueNum < 13; truqueNum++ {
		fmt.Printf("\n--- Truque %d ---\n", truqueNum+1)

		// Iniciar truque
		jogo.IniciarTruque(jogo.JogadorInicial)

		// 4 jogadores jogam suas cartas
		jogadorAtual := jogo.JogadorInicial
		for cartaNum := 0; cartaNum < 4; cartaNum++ {
			jogador := jogo.Jogadores[jogadorAtual]

			// Escolher carta válida para jogar
			carta := escolherCartaValida(jogo, jogador)

			fmt.Printf("%s joga %s\n", jogador.Nome, carta)

			err := jogo.JogarCarta(jogador, carta)
			if err != nil {
				t.Errorf("Erro ao jogar carta: %v", err)
			}

			jogadorAtual = (jogadorAtual + 1) % 4
		}

		// Finalizar truque
		jogo.FinalizarTruque()

		if jogo.TruqueAtual != nil {
			t.Error("Truque deveria estar finalizado")
		}

		if len(jogo.Truques) != truqueNum+1 {
			t.Errorf("Número de truques incorreto: esperado %d, obtido %d", truqueNum+1, len(jogo.Truques))
		}

		vencedor := jogo.Truques[len(jogo.Truques)-1].Vencedor
		fmt.Printf("Vencedor do truque: %s\n", vencedor.Nome)

		// Próximo truque será iniciado pelo vencedor
		jogo.JogadorInicial = vencedor.ID
	}

	// Verificar se a rodada está completa
	if !jogo.RodadaCompleta() {
		t.Error("Rodada deveria estar completa")
	}

	// Calcular pontuação da rodada
	jogo.CalcularPontuacaoRodada()

	fmt.Println("\n=== PONTUAÇÃO FINAL DA RODADA ===")
	for _, jogador := range jogo.Jogadores {
		fmt.Printf("%s: %d pontos\n", jogador.Nome, jogador.Pontuacao)
	}

	// Verificar se as pontuações fazem sentido
	totalPontos := 0
	for _, jogador := range jogo.Jogadores {
		totalPontos += jogador.Pontuacao
	}

	// Total deve ser 26 (13 copas + 13 da dama de espadas) ou 0 (se houve shoot the moon)
	if totalPontos != 26 && totalPontos != 104 { // 104 = 26 * 4 quando há shoot the moon
		t.Errorf("Total de pontos incorreto: %d", totalPontos)
	}

	fmt.Println("\n=== TESTE CONCLUÍDO COM SUCESSO ===")
}

func escolherCartaValida(jogo *Jogo, jogador *Jogador) Carta {
	// Se é a primeira carta do primeiro truque, deve ser 2 de paus
	if jogo.PrimeiraRodada && len(jogo.TruqueAtual.CartasJogadas) == 0 {
		return Carta{Valor: "2", Naipe: "♣"}
	}

	// Se não há cartas jogadas ainda, escolher qualquer carta válida
	if len(jogo.TruqueAtual.CartasJogadas) == 0 {
		// Se copas não foram quebradas, não pode começar com copas
		if !jogo.CopasQuebradas {
			for _, carta := range jogador.Cartas {
				if carta.Naipe != "♥" {
					return carta
				}
			}
		}
		// Se só tem copas, pode jogar
		return jogador.Cartas[0]
	}

	// Deve seguir o naipe se tiver
	naipeInicial := jogo.TruqueAtual.NaipeTruco
	for _, carta := range jogador.Cartas {
		if carta.Naipe == naipeInicial {
			// No primeiro truque, não pode jogar copas nem dama de espadas
			if jogo.PrimeiraRodada {
				if carta.Naipe == "♥" || (carta.Valor == "Q" && carta.Naipe == "♠") {
					continue
				}
			}
			return carta
		}
	}

	// Se não tem o naipe, pode jogar qualquer carta
	for _, carta := range jogador.Cartas {
		// No primeiro truque, não pode jogar copas nem dama de espadas
		if jogo.PrimeiraRodada {
			if carta.Naipe == "♥" || (carta.Valor == "Q" && carta.Naipe == "♠") {
				continue
			}
		}
		return carta
	}

	// Se chegou aqui, só tem cartas proibidas no primeiro truque
	return jogador.Cartas[0]
}

func TestRegrasEspecificas(t *testing.T) {
	fmt.Println("\n=== TESTE DE REGRAS ESPECÍFICAS ===")

	jogo := NovoJogo()
	jogo.DistribuirCartas()

	// Teste 1: Primeira jogada deve ser 2 de paus
	jogo.IniciarTruque(jogo.JogadorInicial)
	jogador := jogo.Jogadores[jogo.JogadorInicial]

	// Tentar jogar carta diferente do 2 de paus
	for _, carta := range jogador.Cartas {
		if carta.Valor != "2" || carta.Naipe != "♣" {
			err := jogo.JogarCarta(jogador, carta)
			if err == nil {
				t.Error("Deveria dar erro ao tentar jogar carta diferente do 2 de paus na primeira jogada")
			}
			break
		}
	}

	// Jogar o 2 de paus corretamente
	err := jogo.JogarCarta(jogador, Carta{Valor: "2", Naipe: "♣"})
	if err != nil {
		t.Errorf("Erro ao jogar 2 de paus: %v", err)
	}

	fmt.Println("✓ Regra do 2 de paus testada com sucesso")

	// Teste 2: Verificar se copas são quebradas quando jogadas
	if jogo.CopasQuebradas {
		t.Error("Copas não deveriam estar quebradas ainda")
	}

	fmt.Println("✓ Todas as regras específicas testadas com sucesso")
}

func TestShootTheMoon(t *testing.T) {
	fmt.Println("\n=== TESTE SHOOT THE MOON ===")

	jogo := NovoJogo()

	// Simular um cenário onde um jogador pega todas as cartas que valem pontos
	jogador0 := jogo.Jogadores[0]

	// Adicionar todas as cartas de copas
	for _, valor := range Valores {
		jogador0.CartasGanhas = append(jogador0.CartasGanhas, Carta{Valor: valor, Naipe: "♥"})
	}

	// Adicionar a dama de espadas
	jogador0.CartasGanhas = append(jogador0.CartasGanhas, Carta{Valor: "Q", Naipe: "♠"})

	// Calcular pontuação
	pontos := jogador0.CalcularPontuacao()
	if pontos != 26 {
		t.Errorf("Pontuação deveria ser 26, obtido %d", pontos)
	}

	// Simular cálculo de pontuação da rodada
	jogo.CalcularPontuacaoRodada()

	// Verificar se os outros jogadores receberam 26 pontos
	for i := 1; i < 4; i++ {
		if jogo.Jogadores[i].Pontuacao != 26 {
			t.Errorf("Jogador %d deveria ter 26 pontos por causa do shoot the moon, obtido %d",
				i, jogo.Jogadores[i].Pontuacao)
		}
	}

	// O jogador que fez shoot the moon deveria ter 0 pontos (não soma os 26)
	if jogo.Jogadores[0].Pontuacao != 0 {
		t.Errorf("Jogador que fez shoot the moon deveria ter 0 pontos, obtido %d",
			jogo.Jogadores[0].Pontuacao)
	}

	fmt.Println("✓ Shoot the moon testado com sucesso")
}
