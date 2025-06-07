class GameUI:
    @staticmethod
    def clear_screen():
        """Limpa a tela (funciona no Linux/macOS/Windows)"""
        import os
        os.system('clear' if os.name == 'posix' else 'cls')
    
    @staticmethod
    def print_separator():
        """Imprime separador visual"""
        print("=" * 60)
    
    @staticmethod
    def print_game_status(trick_num, cards_on_table, scores):
        """Imprime status consolidado do jogo"""
        print(f"\n📋 RODADA {trick_num}")
        if cards_on_table:
            print(f"🃏 Mesa: {cards_on_table}")
        print(f"📊 Pontuações: {scores}")
        GameUI.print_separator()
    
    @staticmethod
    def print_trick_result(winner, cards, points, new_scores):
        """Imprime resultado da rodada de forma limpa"""
        print(f"\n🏆 RESULTADO DA RODADA")
        print(f"📋 Cartas: {cards}")
        print(f"🎯 Vencedor: Player {winner}")
        print(f"💔 Pontos: {points}")
        print(f"📈 Pontuações: {new_scores}")
        GameUI.print_separator()
    
    @staticmethod
    def print_player_turn(player_index):
        """Anuncia vez do jogador"""
        print(f"\n🎯 VEZ DO PLAYER {player_index}")
        GameUI.print_separator()