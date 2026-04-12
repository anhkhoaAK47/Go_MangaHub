package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go_mangahub/manga_hub/pkg/models"
)


func SeedSampleManga(db *sql.DB) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM manga").Scan(&count)
	if count > 0 {
		return err
	}

	// Sample Manga Data
	mangaList := []models.Manga{
	// SHOUNEN
		{ID: "one-piece", Title: "One Piece", Author: "Eiichiro Oda", Genres: []string{"Action", "Adventure", "Shounen"}, Status: "ongoing", TotalChapters: 1100, Description: "A pirate's quest for the ultimate treasure."},
		{ID: "naruto", Title: "Naruto", Author: "Masashi Kishimoto", Genres: []string{"Action", "Ninja", "Shounen"}, Status: "completed", TotalChapters: 700, Description: "A young ninja who seeks recognition."},
		{ID: "bleach", Title: "Bleach", Author: "Tite Kubo", Genres: []string{"Action", "Supernatural", "Shounen"}, Status: "completed", TotalChapters: 686, Description: "High schooler who can see ghosts."},
		{ID: "jujutsu-kaisen", Title: "Jujutsu Kaisen", Author: "Gege Akutami", Genres: []string{"Action", "Supernatural", "Shounen"}, Status: "ongoing", TotalChapters: 250, Description: "Cursed energy and sorcerers."},
		{ID: "demon-slayer", Title: "Demon Slayer", Author: "Koyoharu Gotouge", Genres: []string{"Action", "Demons", "Shounen"}, Status: "completed", TotalChapters: 205, Description: "Fighting demons to save a sister."},
		{ID: "my-hero-academia", Title: "My Hero Academia", Author: "Kohei Horikoshi", Genres: []string{"Action", "Superpowers", "Shounen"}, Status: "ongoing", TotalChapters: 410, Description: "A world where everyone has a quirk."},
		{ID: "attack-on-titan", Title: "Attack on Titan", Author: "Hajime Isayama", Genres: []string{"Action", "Drama", "Shounen"}, Status: "completed", TotalChapters: 139, Description: "Humanity fighting giant titans."},
		{ID: "chainsaw-man", Title: "Chainsaw Man", Author: "Tatsuki Fujimoto", Genres: []string{"Action", "Horror", "Shounen"}, Status: "ongoing", TotalChapters: 150, Description: "A man who becomes a chainsaw devil."},
		{ID: "hunter-x-hunter", Title: "Hunter x Hunter", Author: "Yoshihiro Togashi", Genres: []string{"Action", "Adventure", "Shounen"}, Status: "ongoing", TotalChapters: 400, Description: "Gon's journey to find his father."},
		{ID: "dragon-ball", Title: "Dragon Ball", Author: "Akira Toriyama", Genres: []string{"Action", "Martial Arts", "Shounen"}, Status: "completed", TotalChapters: 519, Description: "Goku's adventures."},
		{ID: "black-clover", Title: "Black Clover", Author: "Yūki Tabata", Genres: []string{"Action", "Magic", "Shounen"}, Status: "ongoing", TotalChapters: 370, Description: "A boy without magic in a magic world."},
		{ID: "spy-x-family", Title: "Spy x Family", Author: "Tatsuya Endo", Genres: []string{"Comedy", "Action", "Shounen"}, Status: "ongoing", TotalChapters: 90, Description: "A spy, an assassin, and a telepath."},
		{ID: "haikyuu", Title: "Haikyuu!!", Author: "Haruichi Furudate", Genres: []string{"Sports", "Volleyball", "Shounen"}, Status: "completed", TotalChapters: 402, Description: "High school volleyball journey."},
		{ID: "blue-lock", Title: "Blue Lock", Author: "Muneyuki Kaneshiro", Genres: []string{"Sports", "Soccer", "Shounen"}, Status: "ongoing", TotalChapters: 240, Description: "Creating the world's best striker."},
		{ID: "dr-stone", Title: "Dr. Stone", Author: "Riichiro Inagaki", Genres: []string{"Sci-Fi", "Adventure", "Shounen"}, Status: "completed", TotalChapters: 232, Description: "Rebuilding civilization with science."},

    // SEINEN
		{ID: "berserk", Title: "Berserk", Author: "Kentaro Miura", Genres: []string{"Action", "Dark Fantasy", "Seinen"}, Status: "ongoing", TotalChapters: 375, Description: "Guts, the Black Swordsman."},
		{ID: "vinland-saga", Title: "Vinland Saga", Author: "Makoto Yukimura", Genres: []string{"Action", "Historical", "Seinen"}, Status: "ongoing", TotalChapters: 205, Description: "Viking epic of revenge and peace."},
		{ID: "vagabond", Title: "Vagabond", Author: "Takehiko Inoue", Genres: []string{"Action", "Historical", "Seinen"}, Status: "hiatus", TotalChapters: 327, Description: "Life of Miyamoto Musashi."},
		{ID: "monster", Title: "Monster", Author: "Naoki Urasawa", Genres: []string{"Thriller", "Mystery", "Seinen"}, Status: "completed", TotalChapters: 162, Description: "Doctor chasing a serial killer."},
		{ID: "20th-century-boys", Title: "20th Century Boys", Author: "Naoki Urasawa", Genres: []string{"Mystery", "Sci-Fi", "Seinen"}, Status: "completed", TotalChapters: 249, Description: "Childhood memories saving the world."},
		{ID: "tokyo-ghoul", Title: "Tokyo Ghoul", Author: "Sui Ishida", Genres: []string{"Horror", "Action", "Seinen"}, Status: "completed", TotalChapters: 143, Description: "Humans living among ghouls."},
		{ID: "kingdom", Title: "Kingdom", Author: "Yasuhisa Hara", Genres: []string{"Action", "Historical", "Seinen"}, Status: "ongoing", TotalChapters: 780, Description: "Unifying ancient China."},
		{ID: "golden-kamuy", Title: "Golden Kamuy", Author: "Satoru Noda", Genres: []string{"Adventure", "Historical", "Seinen"}, Status: "completed", TotalChapters: 314, Description: "Hunt for hidden gold in Hokkaido."},
		{ID: "oshi-no-ko", Title: "Oshi no Ko", Author: "Aka Akasaka", Genres: []string{"Drama", "Mystery", "Seinen"}, Status: "ongoing", TotalChapters: 130, Description: "The dark side of the idol industry."},
		{ID: "kaguya-sama", Title: "Kaguya-sama: Love is War", Author: "Aka Akasaka", Genres: []string{"Comedy", "Romance", "Seinen"}, Status: "completed", TotalChapters: 281, Description: "Genius students in love wars."},
		{ID: "dorohedoro", Title: "Dorohedoro", Author: "Q Hayashida", Genres: []string{"Action", "Fantasy", "Seinen"}, Status: "completed", TotalChapters: 167, Description: "Lizard-headed man seeking his identity."},
		{ID: "goodnight-punpun", Title: "Goodnight Punpun", Author: "Inio Asano", Genres: []string{"Drama", "Psychological", "Seinen"}, Status: "completed", TotalChapters: 147, Description: "Life of a boy named Punpun."},
		{ID: "blue-period", Title: "Blue Period", Author: "Tsubasa Yamaguchi", Genres: []string{"Drama", "Art", "Seinen"}, Status: "ongoing", TotalChapters: 60, Description: "High schooler finding passion in art."},
		{ID: "grand-blue", Title: "Grand Blue Dreaming", Author: "Kenji Inoue", Genres: []string{"Comedy", "Slice of Life", "Seinen"}, Status: "ongoing", TotalChapters: 85, Description: "Diving and drinking comedy."},
		{ID: "parasyte", Title: "Parasyte", Author: "Hitoshi Iwaaki", Genres: []string{"Horror", "Sci-Fi", "Seinen"}, Status: "completed", TotalChapters: 64, Description: "Alien parasites taking over humans."},

    // OTHERS
		{ID: "nana", Title: "Nana", Author: "Ai Yazawa", Genres: []string{"Drama", "Music", "Shoujo"}, Status: "hiatus", TotalChapters: 84, Description: "Two girls named Nana meeting in Tokyo."},
		{ID: "fruits-basket", Title: "Fruits Basket", Author: "Natsuki Takaya", Genres: []string{"Romance", "Drama", "Shoujo"}, Status: "completed", TotalChapters: 136, Description: "The Soma family curse."},
		{ID: "sailor-moon", Title: "Sailor Moon", Author: "Naoko Takeuchi", Genres: []string{"Fantasy", "Magical Girl", "Shoujo"}, Status: "completed", TotalChapters: 60, Description: "Fighting evil by moonlight."},
		{ID: "chihayafuru", Title: "Chihayafuru", Author: "Yuki Suetsugu", Genres: []string{"Sports", "Drama", "Josei"}, Status: "completed", TotalChapters: 247, Description: "The world of competitive Karuta."},
		{ID: "wotakoi", Title: "Wotakoi: Love is Hard for Otaku", Author: "Fujita", Genres: []string{"Comedy", "Romance", "Josei"}, Status: "completed", TotalChapters: 60, Description: "Otaku office romance."},
		{ID: "honey-and-clover", Title: "Honey and Clover", Author: "Chica Umino", Genres: []string{"Drama", "Romance", "Josei"}, Status: "completed", TotalChapters: 71, Description: "Life at an art college."},
		{ID: "cardcaptor-sakura", Title: "Cardcaptor Sakura", Author: "CLAMP", Genres: []string{"Adventure", "Fantasy", "Shoujo"}, Status: "completed", TotalChapters: 50, Description: "Collecting magical Clow Cards."},
		{ID: "blue-spring-ride", Title: "Blue Spring Ride", Author: "Io Sakisaka", Genres: []string{"Romance", "Drama", "Shoujo"}, Status: "completed", TotalChapters: 53, Description: "Reuniting with a first love."},
		{ID: "horimiya", Title: "Horimiya", Author: "HERO", Genres: []string{"Romance", "Comedy", "Shounen"}, Status: "completed", TotalChapters: 125, Description: "Hidden sides of popular students."},
		{ID: "re-life", Title: "ReLife", Author: "Yayoiso", Genres: []string{"Drama", "Slice of Life", "Seinen"}, Status: "completed", TotalChapters: 222, Description: "A 27-year-old reliving high school."},
	}

	// insert into database
	query := `INSERT INTO manga (id, title, author, genres, status, total_chapters, description) VALUES (?, ?, ?, ?, ?, ?, ?)`


	for _, m := range mangaList {
		// Convert genre slice to JSON
		genreJSON, _ := json.Marshal(m.Genres)

		 _, err := db.Exec(query, 
        m.ID, 
        m.Title, 
        m.Author, 
        string(genreJSON), 
        m.Status, 
        m.TotalChapters, 
        m.Description,
    )
		if err != nil {
			fmt.Printf("Error seeding manga %s: %s\n", m.Title, err)
		}
	}
	fmt.Println("Manga seeded successfully!")
	return nil
}