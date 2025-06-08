import json
import time
import sys
from network import NetworkManager
from gameui import GameUI

class Protocol:
    def __init__(self, network_manager, game_logic):
        self.network = network_manager
        self.game = game_logic
        self.network.set_message_handler(self.handle_message)
    
    def handle_message(self, message, addr):
        """Processa mensagens recebidas da rede"""
        # Se for apenas o token, processa diretamente
        if message == "TOKEN":
            if self.game.receive_token():  # Só imprime se realmente recebeu
                GameUI.print_player_turn(self.game.player_index)
            return
        
        # Tenta processar como JSON
        try:
            data = json.loads(message)
            msg_type = data.get("type")
            
            # Mapeamento de tipos de mensagem para métodos
            message_handlers = {
                "CONNECT": self.process_connect_message,
                "START_GAME": self.process_start_game_message,
                "GAME": self.process_game_message,
                "END_TRICK": self.process_end_trick_message,
                "SCORES": self.process_scores_message,
                "NEW_HAND": self.process_new_hand_message,
                "GAME_END": self.process_game_end_message
            }
            
            handler = message_handlers.get(msg_type)
            if handler:
                handler(data)
            else:
                print(f"⚠️ Tipo de mensagem desconhecido: {msg_type}")
        
        except json.JSONDecodeError:
            print(f"⚠️ Erro ao decodificar JSON: {message}")

    def process_connect_message(self, data):
        """Processa mensagem de conexão de jogador"""
        player_id = data.get("player")
        
        # Apenas o host (player 0) processa conexões
        if self.game.is_host():
            connected_count = self.game.add_connected_player(player_id)
            print(f"🔗 Player {player_id} conectado! ({connected_count}/4)")
            
            # Se todos os 4 jogadores estão conectados, inicia o jogo
            if self.game.all_players_connected():
                print("🎉 Todos os jogadores conectados! Iniciando jogo...")
                self.start_game_as_host()

    def process_start_game_message(self, data):
        """Processa mensagem de início de jogo"""
        hands = data.get("hands", [])
        
        if self.game.player_index < len(hands):
            self.game.set_player_hand(hands[self.game.player_index])
            self.game.start_game()
                    
            # Verifica se este jogador tem o 2♣ e deve começar
            if self.game.has_two_of_clubs():
                self.send_token_to_self()
            else:
                self.game.token = False
                print(f"🔄 Player {self.game.player_index} não tem o 2♣, esperando o próximo jogador.")
    
    def process_game_message(self, data):
        """Processa mensagens de jogo (jogadas)"""
        action = data.get("action")
        
        if action == "PLAY":
            card = data.get("card")
            player = data.get("player")
            
            # Adiciona a carta jogada às cartas da rodada atual
            self.game.add_card_to_trick(card, player)
            
            # Interface mais limpa
            if player == self.game.player_index:
                print(f"✅ Você jogou: {card}")
            else:
                print(f"🃏 Player {player} jogou: {card}")
            
            # Mostra estado atual da mesa
            cards_on_table = self.game.get_current_trick_cards()
            print(f"📋 Mesa: {cards_on_table} ({len(cards_on_table)}/4)")
            
            # Se todos os 4 jogadores jogaram, termina a rodada
            if self.game.is_trick_complete():
                self.end_trick()

    def process_end_trick_message(self, data):
        """Processa mensagem de fim de rodada"""
        winner = data.get("winner")
        points = data.get("points")
        scores = data.get("scores")
        
        # Atualiza estado
        if scores:
            self.game.update_scores(scores)
        
        self.game.reset_trick()
        
        if self.game.is_hand_complete():
            self.game.check_game_end()
        else:
            if self.game.player_index == winner:
                self.send_token_to_self()
            else:
                self.game.token = False
    
    def process_scores_message(self, data):
        """Processa atualização de pontuações"""
        scores = data.get("scores")
        if scores:
            self.game.update_scores(scores)
            print(f"📊 Pontuações atualizadas: {scores}")

    def process_new_hand_message(self, data):
        """Processa início de nova mão"""
        hands = data.get("hands", [])
        
        if self.game.player_index < len(hands):
            self.game.start_new_hand(hands[self.game.player_index])
            
            print(f"🆕 Nova mão iniciada! Suas cartas: {sorted(self.game.player_hand, key=lambda x: (self.game.get_card_suit(x), self.game.get_card_value(x)))}")
            
            # Verifica se este jogador tem o 2♣ e deve começar
            if self.game.has_two_of_clubs():
                self.send_token_to_self()
            else:
                self.game.token = False
                print(f"🔄 Player {self.game.player_index} não tem o 2♣, esperando o próximo jogador.")

    def process_game_end_message(self, data):
        """Processa fim de jogo"""
        winner = data.get("winner")
        final_scores = data.get("final_scores")
        
        self.game.end_game(winner, final_scores)
        
        print(f"\n🎉 JOGO TERMINADO!")
        print(f"🏆 Player {winner} venceu com {final_scores[winner]} pontos!")
        print(f"📊 Pontuações finais: {final_scores}")
        
        # Encerra o programa após alguns segundos
        time.sleep(3)
        self.network.close()
        sys.exit(0)

    # MÉTODOS DE ENVIO DE MENSAGENS
    def send_connect_message(self):
        """Envia mensagem de conexão para o host"""
        if not self.game.is_host():
            connect_message = {
                "type": "CONNECT",
                "player": self.game.player_index
            }
            self.network.send_message(json.dumps(connect_message), 0)
            print("📡 Conexão anunciada para o host")

    def send_play_card_message(self, card):
        """Envia mensagem de jogada de carta"""
        message = {
            "type": "GAME",
            "action": "PLAY",
            "card": card,
            "player": self.game.player_index
        }
        self.network.send_to_all(json.dumps(message))

    def send_end_trick_message(self, winner, points, scores):
        """Envia mensagem de fim de rodada"""
        end_trick_message = {
            "type": "END_TRICK",
            "winner": winner,
            "points": points,
            "scores": scores,
            "trick": self.game.current_trick + 1
        }
        self.network.send_to_all(json.dumps(end_trick_message))

    def send_game_end_message(self, winner, final_scores):
        """Envia mensagem de fim de jogo"""
        game_end_message = {
            "type": "GAME_END",
            "winner": winner,
            "final_scores": final_scores
        }
        self.network.send_to_all(json.dumps(game_end_message))

    def send_new_hand_message(self, hands):
        """Envia mensagem de nova mão"""
        start_message = {
            "type": "NEW_HAND",
            "hands": hands
        }
        self.network.send_to_all(json.dumps(start_message))

    def send_token_to_self(self):
        """Envia token para si mesmo"""
        # Só envia se não tiver o token já
        if not self.game.token:
            self.network.send_message("TOKEN", self.game.player_index)
    
    def pass_token_to_next(self):
        """Passa token para o próximo jogador"""
        self.game.token = False
        self.network.pass_token(self.network.next_node_index)

    # MÉTODOS DE COORDENAÇÃO
    def start_game_as_host(self):
        """Inicia o jogo como host"""
        print("🎲 Host distribuindo cartas...")
        hands = self.game.deal_cards()
        self.game.set_player_hand(hands[0])
        
        # Envia as cartas para todos os jogadores
        start_message = {
            "type": "START_GAME",
            "hands": hands
        }
        self.network.send_to_all(json.dumps(start_message))
        
        self.game.start_game()
        
        # O jogador com 2♣ recebe o token
        if self.game.has_two_of_clubs():
            self.send_token_to_self()
        else:
            self.game.token = False

    def end_trick(self):
        """Finaliza uma rodada"""
        winner, points = self.game.calculate_trick_result()
        
        # Interface melhorada
        GameUI.print_trick_result(
            winner, 
            self.game.get_current_trick_cards(), 
            points, 
            self.game.players_scores.copy()
        )
        
        self.game.next_trick()
        
        # APENAS O VENCEDOR envia a mensagem
        if self.game.player_index == winner:
            self.send_end_trick_message(winner, points, self.game.players_scores.copy())
            print(f"🎯 Você ganhou a rodada! É sua vez de jogar.")
            self.send_token_to_self()
            
    def play_card(self, card):
        """Processa jogada de carta do jogador local"""
        if self.game.can_play_card(card):
            will_complete = self.game.will_complete_trick()
            self.game.remove_card_from_hand(card)
            self.send_play_card_message(card)
            self.game.token = False

            if not will_complete:
                # Se ainda não completou a rodada, passa o token para o próximo jogador
                self.pass_token_to_next()
        else:        
            print("❌ Você não tem essa carta na mão!")

    def initialize_connection(self):
        """Inicializa a conexão do jogador"""
        self.game.initialize()
        
        if self.game.is_host():
            print(f"🎮 Player {self.game.player_index} (Host) - Aguardando outros jogadores...")
        else:
            print(f"🎮 Player {self.game.player_index} - Conectando ao jogo...")
            self.send_connect_message()
            print("⏳ Aguardando início do jogo...")