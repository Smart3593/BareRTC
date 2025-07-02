package barertc

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"git.kirsle.net/apps/barertc/pkg/config"
	ourjwt "git.kirsle.net/apps/barertc/pkg/jwt"
	"git.kirsle.net/apps/barertc/pkg/log"
	"git.kirsle.net/apps/barertc/pkg/messages"
	"github.com/golang-jwt/jwt/v4"
	"github.com/mattn/go-shellwords"
)

// ProcessCommand parses a chat message for "/commands"
func (s *Server) ProcessCommand(sub *Subscriber, msg messages.Message) bool {
	if len(msg.Message) == 0 || msg.Message[0] != '/' {
		return false
	}

	// Line begins with a slash, parse it apart.
	words, err := shellwords.Parse(msg.Message)
	if err != nil {
		log.Error("ProcessCommands: parsing shell words: %s", err)
		return false
	} else if len(words) == 0 {
		return false
	}

	// Moderator commands.
	if sub.JWTClaims != nil && sub.JWTClaims.IsAdmin {
		switch words[0] {
		case "/kick":
			s.KickCommand(words, sub)
			return true
		case "/ban":
			s.BanCommand(words, sub)
			return true
		case "/unban":
			s.UnbanCommand(words, sub)
			return true
		case "/bans":
			s.BansCommand(words, sub)
			return true
		case "/nsfw":
			s.NSFWCommand(words, sub)
			return true
		case "/cut":
			s.CutCommand(words, sub)
			return true
		case "/unmute-all":
			s.UnmuteAllCommand(words, sub)
			return true
		case "/help":
			sub.ChatServer(RenderMarkdown("Les commandes de modération les plus courantes sur le chat sont :\n\n" +
				"* `/kick <nom_utilisateur>` pour exclure du chat\n" +
				"* `/ban <nom_utilisateur> <durée>` pour bannir du chat (durée par défaut : 24 heures)\n" +
				"* `/unban <nom_utilisateur>` pour lever le bannissement d'un utilisateur\n" +
				"* `/bans` pour lister les utilisateurs bannis et leur date d'expiration\n" +
				"* `/nsfw <nom_utilisateur>` pour marquer sa caméra comme NSFW\n" +
				"* `/cut <nom_utilisateur>` pour lui demander d'éteindre sa caméra\n" +
				"* `/help` pour afficher ce message\n" +
				"* `/help-advanced` pour afficher les commandes avancées d'administration\n\n" +
				"Note : la citation de style shell est prise en charge, si un nom d'utilisateur contient un espace, citez tout le nom, par exemple : `/kick \"utilisateur 2\"`"),
			)
			return true
		case "/help-advanced":
			sub.ChatServer(RenderMarkdown("Les commandes suivantes sont **dangereuses** et ne doivent être utilisées que si vous savez ce que vous faites :\n\n" +
				"* `/op <nom_utilisateur>` pour accorder les droits d'opérateur à un utilisateur\n" +
				"* `/deop <nom_utilisateur>` pour retirer les droits d'opérateur à un utilisateur\n" +
				"* `/shutdown` pour redémarrer le serveur de chat\n" +
				"* `/kickall` pour exclure TOUT LE MONDE et forcer la reconnexion\n" +
				"* `/reconfigure` pour recharger dynamiquement le fichier de configuration du serveur\n" +
				"* `/help-advanced` pour afficher ce message"),
			)
			return true
		case "/shutdown":
			s.Broadcast(messages.Message{
				Action:   messages.ActionError,
				Username: "ChatServer",
				Message:  "The chat server is going down for a reboot NOW!",
			})
			os.Exit(1)
			return true
		case "/kickall":
			s.KickAllCommand()
			return true
		case "/reconfigure":
			s.ReconfigureCommand(sub)
			return true
		case "/op":
			s.OpCommand(words, sub)
			return true
		case "/deop":
			s.DeopCommand(words, sub)
			return true
		}

	}

	// Not handled.
	return false
}

// NSFWCommand handles the `/nsfw` operator command.
func (s *Server) NSFWCommand(words []string, sub *Subscriber) {
	if len(words) == 1 {
		sub.ChatServer("Utilisation : `/nsfw nom_utilisateur` pour ajouter le drapeau NSFW à sa caméra.")
	}
	username := strings.TrimPrefix(words[1], "@")
	other, err := s.GetSubscriber(username)
	if err != nil {
		sub.ChatServer("/nsfw : nom d'utilisateur introuvable : %s", username)
	} else {
		// Sanity check that the target user is presently on a blue camera.
		if !(other.VideoStatus&messages.VideoFlagActive == messages.VideoFlagActive) {
			sub.ChatServer("/nsfw : la caméra de %s n'était pas activée.", username)
			return
		} else if other.VideoStatus&messages.VideoFlagNSFW == messages.VideoFlagNSFW {
			sub.ChatServer("/nsfw : la caméra de %s était déjà marquée comme explicite.", username)
			return
		}

		// The message to deliver to the target.
		var message = "Petit rappel amical de marquer votre caméra comme 'Explicite' en utilisant le bouton en haut " +
			"de la page si vous comptez avoir un comportement sexuel sur la webcam.<br><br>"

		// If the admin who marked it was previously booted
		if other.Boots(sub.Username) {
			message += "Votre caméra a été détectée comme présentant une activité 'Explicite' et a été marquée pour vous."
		} else {
			message += fmt.Sprintf("Votre caméra a été marquée comme Explicite pour vous par @%s", sub.Username)
		}

		other.ChatServer(message)
		other.VideoStatus |= messages.VideoFlagNSFW
		other.SendMe()
		s.SendWhoList()
		sub.ChatServer("%s a maintenant sa caméra marquée comme Explicite", username)

		// Send an admin report to your main website.
		if err := PostWebhookReport(WebhookRequestReport{
			FromUsername:  sub.Username,
			AboutUsername: username,
			Channel:       "n/a",
			Timestamp:     time.Now().Format(time.RFC3339),
			Reason:        "Commande NSFW émise",
			Message:       fmt.Sprintf("L'administrateur @%s marque la webcam en rouge pour l'utilisateur @%s", sub.Username, username),
			Comment:       "Un administrateur a marqué leur webcam comme explicite.",
		}); err != nil {
			log.Error("Error delivering a report to your website about the /nsfw command by %s: %s", sub.Username, err)
		}
	}
}

// CutCommand handles the `/cut` operator command (force a user's camera to turn off).
func (s *Server) CutCommand(words []string, sub *Subscriber) {
	if len(words) == 1 {
		sub.ChatServer("Utilisation : `/cut nom_utilisateur` pour éteindre sa caméra.")
	}
	username := strings.TrimPrefix(words[1], "@")
	other, err := s.GetSubscriber(username)
	if err != nil {
		sub.ChatServer("/cut : nom d'utilisateur introuvable : %s", username)
	} else {
		// Sanity check that the target user is presently on a blue camera.
		if !(other.VideoStatus&messages.VideoFlagActive == messages.VideoFlagActive) {
			sub.ChatServer("/cut : la caméra de %s n'était pas activée.", username)
			return
		}

		other.SendCut()
		sub.ChatServer("%s a reçu la consigne d'éteindre sa caméra.", username)
	}
}

// UnmuteAllCommand handles the `/unmute-all` operator command (remove all mutes for the current user).
//
// It enables an operator to see public messages from any user who muted/blocked them. Note: from the
// other side of the mute, the operator's public messages may still be hidden from those users.
//
// It is useful for an operator chatbot if you want users to be able to block it but still retain the
// bot's ability to moderate public channel messages, and send warnings in DMs to misbehaving users
// even despite a mute being in place.
func (s *Server) UnmuteAllCommand(words []string, sub *Subscriber) {
	count := len(sub.muted)
	sub.muted = map[string]struct{}{}
	sub.unblockable = true
	sub.ChatServer("Votre mise en sourdine sur %d utilisateurs a été levée.", count)
	s.SendWhoList()
}

// KickCommand handles the `/kick` operator command.
func (s *Server) KickCommand(words []string, sub *Subscriber) {
	if len(words) == 1 {
		sub.ChatServer(RenderMarkdown(
			"Utilisation : `/kick nom_utilisateur` pour exclure l'utilisateur du salon de discussion.\n\nNote : si le nom d'utilisateur contient des espaces, citez le nom (style shell), `/kick \"utilisateur 2\"`",
		))
		return
	}
	username := strings.TrimPrefix(words[1], "@")
	other, err := s.GetSubscriber(username)
	if err != nil {
		sub.ChatServer("/kick : nom d'utilisateur introuvable : %s", username)
	} else if other.Username == sub.Username {
		sub.ChatServer("/kick : vouliez-vous vraiment vous exclure vous-même ?")
	} else {
		s.KickUser(other, fmt.Sprintf("Vous avez été exclu du salon de discussion par %s", sub.Username), false)
		sub.ChatServer("%s a été exclu du salon", username)
	}
}

// KickAllCommand kicks everybody out of the room.
func (s *Server) KickAllCommand() {

	// If we have JWT enabled and a landing page, link users to it.
	if config.Current.JWT.Enabled && config.Current.JWT.LandingPageURL != "" {
		s.Broadcast(messages.Message{
			Action:   messages.ActionError,
			Username: "ChatServer",
			Message: fmt.Sprintf(
				"<strong>Attention :</strong> L'opérateur du chat vous demande de vous reconnecter au salon. "+
					"C'est probablement parce qu'une nouvelle fonctionnalité a été lancée et nécessite de recharger la page. "+
					"Vous pouvez actualiser l'onglet ou <a href=\"%s\">cliquer ici</a> pour revenir dans le salon.",
				config.Current.JWT.LandingPageURL,
			),
		})
	} else {
		s.Broadcast(messages.Message{
			Action:   messages.ActionError,
			Username: "ChatServer",
			Message: "<strong>Attention :</strong> L'opérateur du chat a exclu tout le monde du salon. Généralement, cela " +
				"signifie qu'une nouvelle fonctionnalité du chat a été lancée et que vous devez recharger la page pour qu'elle " +
				"fonctionne correctement.",
		})
	}

	// Kick everyone off.
	s.Broadcast(messages.Message{
		Action: messages.ActionKick,
	})

	// Disconnect everybody.
	for _, sub := range s.IterSubscribers() {
		if !sub.authenticated {
			continue
		}

		sub.authenticated = false
		sub.Username = ""
	}
}

// BanCommand handles the `/ban` operator command.
func (s *Server) BanCommand(words []string, sub *Subscriber) {
	if len(words) == 1 {
		sub.ChatServer(RenderMarkdown(
			"Utilisation : `/ban nom_utilisateur` pour exclure l'utilisateur du salon pendant 24 heures (par défaut).\n\n" +
				"Définissez une autre durée (en heures) comme : `/ban nom_utilisateur 2` pour un bannissement de 2 heures.",
		))
		return
	}

	// Parse the command.
	var (
		username = strings.TrimPrefix(words[1], "@")
		duration = 24 * time.Hour
	)
	if len(words) >= 3 {
		if dur, err := strconv.Atoi(words[2]); err == nil {
			duration = time.Duration(dur) * time.Hour
		}
	}

	log.Info("Operator %s bans %s for %d hours", sub.Username, username, duration/time.Hour)

	// Add them to the ban list.
	s.BanUser(username, duration)

	// If the target user is currently online, disconnect them and broadcast the ban to everybody.
	if other, err := s.GetSubscriber(username); err == nil {
		s.KickUser(other, fmt.Sprintf("Vous avez été banni du salon de discussion par %s. Vous pourrez revenir après %d heures.", sub.Username, duration/time.Hour), true)
	}

	sub.ChatServer("%s a été banni du salon pour %d heures.", username, duration/time.Hour)
}

// UnbanCommand handles the `/unban` operator command.
func (s *Server) UnbanCommand(words []string, sub *Subscriber) {
	if len(words) == 1 {
		sub.ChatServer(RenderMarkdown(
			"Utilisation : `/unban nom_utilisateur` pour lever le bannissement d'un utilisateur et lui permettre de revenir dans le salon.",
		))
		return
	}

	// Parse the command.
	var username = strings.TrimPrefix(words[1], "@")

	if UnbanUser(username) {
		sub.ChatServer("Le bannissement de %s a été levé.", username)
	} else {
		sub.ChatServer("/unban : l'utilisateur %s n'a pas été trouvé comme banni. Essayez `/bans` pour voir les utilisateurs actuellement bannis.", username)
	}
}

// BansCommand handles the `/bans` operator command.
func (s *Server) BansCommand(words []string, sub *Subscriber) {
	result := StringifyBannedUsers()
	sub.ChatServer(
		RenderMarkdown("La liste des utilisateurs bannis comprend actuellement :\n\n" + result),
	)
}

// ReconfigureCommand handles the `/reconfigure` operator command.
func (s *Server) ReconfigureCommand(sub *Subscriber) {
	// Reload the settings.
	if err := config.LoadSettings(); err != nil {
		sub.ChatServer("Erreur lors du rechargement de la configuration du serveur : %s", err)
		return
	}

	sub.ChatServer("Le fichier de configuration du serveur a été rechargé avec succès !")
}

// OpCommand handles the `/op` operator command.
func (s *Server) OpCommand(words []string, sub *Subscriber) {
	if len(words) == 1 {
		sub.ChatServer(RenderMarkdown(
			"Utilisation : `/op nom_utilisateur` pour accorder temporairement les droits d'opérateur à un utilisateur.",
		))
		return
	}

	// Parse the command.
	var username = strings.TrimPrefix(words[1], "@")
	if other, err := s.GetSubscriber(username); err != nil {
		sub.ChatServer("/op : utilisateur %s introuvable.", username)
	} else {
		if other.JWTClaims == nil {
			other.JWTClaims = &ourjwt.Claims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: username,
				},
			}
		}
		other.JWTClaims.IsAdmin = true

		// Send everyone the Who List.
		s.SendWhoList()

		sub.ChatServer("Les droits d'opérateur ont été accordés à %s", username)
	}
}

// DeopCommand handles the `/deop` operator command.
func (s *Server) DeopCommand(words []string, sub *Subscriber) {
	if len(words) == 1 {
		sub.ChatServer(RenderMarkdown(
			"Utilisation : `/deop nom_utilisateur` pour retirer les droits d'opérateur à un utilisateur.",
		))
		return
	}

	// Parse the command.
	var username = strings.TrimPrefix(words[1], "@")
	if other, err := s.GetSubscriber(username); err != nil {
		sub.ChatServer("/deop : utilisateur %s introuvable.", username)
	} else {
		if other.JWTClaims == nil {
			other.JWTClaims = &ourjwt.Claims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: username,
				},
			}
		}
		other.JWTClaims.IsAdmin = false

		// Send everyone the Who List.
		s.SendWhoList()

		sub.ChatServer("Les droits d'opérateur ont été retirés à %s", username)
	}
}
