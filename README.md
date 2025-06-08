# Descrição do Projeto

Implementar o jogo copas em uma rede em anel com 4 máquinas

- Criar uma rede em anel com 4 máquinas usando socket DGRAM
- O Controle de acesso a rede deve ser feito por passagem de bastão
- Todas as mensagens devem dar a volta toda pelo anel
- O bastão não é temporizado
- As mensagens não são temporizadas
- Utilizar linguagem Python
- Utilizar o protocolo UDP
- Não é necessário timeout
- O protocolo pode ser de livre escolha

# Como executar
1. Certifique-se de ter o Python instalado.

2. Para executar em localHost, abra 4 terminais diferentes e execute:
   ```bash
   python main.py <player_id> localhost [porta]
   ```
   Onde `<player_id>` pode ser 0, 1, 2 ou 3, representando cada jogador.

3. Para executar em rede, substitua `localhost` pelo IP da máquina onde o servidor está rodando.

# 🎯 Regras do Jogo Copas (Sem Passar Cartas)

## 🔸 Objetivo do Jogo
- Evitar pegar cartas de copas (♥) e a dama de espadas (♠Q), que valem pontos negativos.
- O jogo termina quando algum jogador atinge ou ultrapassa **100 pontos**.
- Vence quem tiver a **menor pontuação**.

---

## 🃏 Configuração
- Baralho padrão de 52 cartas (sem coringas).
- 4 jogadores.
- Cada jogador recebe 13 cartas.

---

## 🔸 Ordem de Jogada
- Quem tiver o **2 de paus (♣2)** começa a primeira rodada.
- Na primeira rodada, é **obrigatório começar com o 2 de paus (♣2)**.
- Depois disso, os jogadores seguem no sentido **horário**.

---

## 🔸 Regras das Rodadas (Truques)
1. O jogador que inicia joga uma carta de qualquer naipe válido (com restrições abaixo).
2. Os outros jogadores, na ordem, devem:
   - Jogar uma carta do **mesmo naipe** se tiverem.
   - Se não tiverem, podem jogar qualquer carta (**restrição sobre copas abaixo**).
3. **Copas (♥) não podem ser jogadas até que sejam "quebradas"**, ou seja, até que algum jogador jogue uma carta de copas por não ter o naipe pedido.
   - **Exceção:** Se o jogador não tiver nenhuma carta de outro naipe, pode jogar copas mesmo antes de serem quebradas.
4. Na **primeira rodada (quando começa com ♣2)**:
   - **Não é permitido jogar cartas de copas (♥) nem a dama de espadas (♠Q)**.
   - Se o jogador não tiver paus, deve jogar qualquer outra carta que **não seja ♥ nem ♠Q**.

---

## 🔸 Quem Ganha o Truque
- Vence o truque quem jogou a carta **mais alta do naipe que iniciou a rodada**.
- Quem vence o truque é quem começa o próximo.

---

## 🔸 Pontuação das Cartas
- Cada carta de **copas (♥)** vale **+1 ponto**.
- A **dama de espadas (♠Q)** vale **+13 pontos**.
- Todas as outras cartas valem **0 pontos**.

---

## 🔸 Shoot the Moon (Varredura)
- Se um jogador pegar **todas as 13 cartas de copas (♥) e a dama de espadas (♠Q)** na mesma rodada:
   - Ao invés de receber 26 pontos, o jogador escolhe:
     - **Todos os outros jogadores recebem +26 pontos.**
- (Em algumas variações, o jogador pode escolher não fazer isso, mas na regra padrão isso é obrigatório.)

---

## 🔸 Fim do Jogo
- Quando algum jogador atinge ou ultrapassa **100 pontos**, o jogo termina.
- O jogador com a **menor pontuação vence**.

---
