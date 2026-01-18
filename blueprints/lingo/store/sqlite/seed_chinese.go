package sqlite

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// ChineseVocab contains vocabulary for Chinese lessons
type ChineseVocab struct {
	Word        string
	Pinyin      string
	Translation string
	POS         string // Part of speech
}

// ChineseUnit defines a unit with skills and vocabulary
type ChineseUnit struct {
	Title       string
	Description string
	Skills      []ChineseSkill
}

// ChineseSkill defines a skill with vocabulary
type ChineseSkill struct {
	Name     string
	IconName string
	Vocab    []ChineseVocab
}

// SeedChineseCourse creates the Chinese for English Speakers course with full content
func (s *Store) SeedChineseCourse(ctx context.Context) error {
	courseID := uuid.New().String()

	// Create the course
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO courses (id, from_language_id, learning_language_id, title, description, total_units, cefr_level, enabled)
		VALUES (?, 'en', 'zh', 'Chinese for English Speakers', 'Learn Mandarin Chinese from scratch with pinyin support', 50, 'A1', 1)
	`, courseID)
	if err != nil {
		return fmt.Errorf("insert Chinese course: %w", err)
	}

	units := getChineseUnits()

	for unitPos, unit := range units {
		unitID := uuid.New().String()
		_, err := s.db.ExecContext(ctx, `
			INSERT INTO units (id, course_id, position, title, description, guidebook_content)
			VALUES (?, ?, ?, ?, ?, ?)
		`, unitID, courseID, unitPos+1, unit.Title, unit.Description, fmt.Sprintf("Welcome to %s! This unit covers essential vocabulary.", unit.Title))
		if err != nil {
			return fmt.Errorf("insert unit %s: %w", unit.Title, err)
		}

		for skillPos, skill := range unit.Skills {
			skillID := uuid.New().String()
			_, err := s.db.ExecContext(ctx, `
				INSERT INTO skills (id, unit_id, position, name, icon_name, levels, lexemes_count)
				VALUES (?, ?, ?, ?, ?, 5, ?)
			`, skillID, unitID, skillPos+1, skill.Name, skill.IconName, len(skill.Vocab))
			if err != nil {
				return fmt.Errorf("insert skill %s: %w", skill.Name, err)
			}

			// Create lexemes for this skill
			lexemeIDs := make([]string, 0, len(skill.Vocab))
			for _, vocab := range skill.Vocab {
				lexemeID := uuid.New().String()
				_, err := s.db.ExecContext(ctx, `
					INSERT INTO lexemes (id, course_id, word, translation, pos, audio_url, example_sentence, example_translation)
					VALUES (?, ?, ?, ?, ?, ?, ?, ?)
				`, lexemeID, courseID, vocab.Word, vocab.Translation, vocab.POS,
					"", // audio_url - would be generated
					"", // example_sentence
					"", // example_translation
				)
				if err != nil {
					return fmt.Errorf("insert lexeme %s: %w", vocab.Word, err)
				}
				lexemeIDs = append(lexemeIDs, lexemeID)
			}

			// Create 5 lessons (one per crown level)
			for level := 1; level <= 5; level++ {
				lessonID := uuid.New().String()
				_, err := s.db.ExecContext(ctx, `
					INSERT INTO lessons (id, skill_id, level, position, exercise_count)
					VALUES (?, ?, ?, 1, 15)
				`, lessonID, skillID, level)
				if err != nil {
					return fmt.Errorf("insert lesson level %d: %w", level, err)
				}

				// Generate exercises for this lesson
				if err := s.seedChineseExercises(ctx, lessonID, skill.Vocab, level); err != nil {
					return fmt.Errorf("seed exercises for lesson %d: %w", level, err)
				}
			}
		}
	}

	return nil
}

func (s *Store) seedChineseExercises(ctx context.Context, lessonID string, vocab []ChineseVocab, level int) error {
	// Exercise distribution based on level
	// Level 1: Recognition focus (easier)
	// Level 5: Production focus (harder)

	exerciseConfigs := []struct {
		Type       string
		Count      int
		Difficulty int
	}{
		{"multiple_choice", 4 - (level / 2), 1},
		{"translation", 3 + (level / 2), level},
		{"word_bank", 3, level},
		{"listening", 2, level},
		{"fill_blank", 2, level},
		{"match_pairs", 1, level},
	}

	exerciseIndex := 0
	for _, config := range exerciseConfigs {
		for i := 0; i < config.Count && exerciseIndex < 15; i++ {
			word := vocab[exerciseIndex%len(vocab)]
			ex := createChineseExercise(config.Type, word, vocab, config.Difficulty)

			choicesJSON, _ := json.Marshal(ex.Choices)
			hintsJSON, _ := json.Marshal(ex.Hints)

			_, err := s.db.ExecContext(ctx, `
				INSERT INTO exercises (id, lesson_id, type, prompt, correct_answer, choices, audio_url, image_url, hints, difficulty)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			`, uuid.New().String(), lessonID, ex.Type, ex.Prompt, ex.CorrectAnswer,
				string(choicesJSON), ex.AudioURL, ex.ImageURL, string(hintsJSON), ex.Difficulty)
			if err != nil {
				return fmt.Errorf("insert exercise: %w", err)
			}
			exerciseIndex++
		}
	}

	return nil
}

type exerciseData struct {
	Type          string
	Prompt        string
	CorrectAnswer string
	Choices       []string
	AudioURL      string
	ImageURL      string
	Hints         []string
	Difficulty    int
}

func createChineseExercise(exType string, word ChineseVocab, allVocab []ChineseVocab, difficulty int) exerciseData {
	switch exType {
	case "multiple_choice":
		// Select correct meaning
		choices := []string{word.Translation}
		for _, v := range allVocab {
			if v.Word != word.Word && len(choices) < 4 {
				choices = append(choices, v.Translation)
			}
		}
		// Add generic wrong answers if needed
		wrongAnswers := []string{"water", "fire", "earth", "wind", "hello", "goodbye"}
		for _, w := range wrongAnswers {
			if len(choices) >= 4 {
				break
			}
			found := false
			for _, c := range choices {
				if c == w {
					found = true
					break
				}
			}
			if !found {
				choices = append(choices, w)
			}
		}
		return exerciseData{
			Type:          "multiple_choice",
			Prompt:        fmt.Sprintf("What does \"%s\" (%s) mean?", word.Word, word.Pinyin),
			CorrectAnswer: word.Translation,
			Choices:       shuffleStrings(choices),
			Hints:         []string{fmt.Sprintf("Pinyin: %s", word.Pinyin)},
			Difficulty:    difficulty,
		}

	case "translation":
		// Translate from Chinese to English
		choices := []string{word.Translation}
		for _, v := range allVocab {
			if v.Word != word.Word && len(choices) < 4 {
				choices = append(choices, v.Translation)
			}
		}
		for len(choices) < 4 {
			choices = append(choices, "unknown")
		}
		return exerciseData{
			Type:          "translation",
			Prompt:        fmt.Sprintf("Translate: %s (%s)", word.Word, word.Pinyin),
			CorrectAnswer: word.Translation,
			Choices:       shuffleStrings(choices),
			Hints:         []string{fmt.Sprintf("This word is a %s", word.POS)},
			Difficulty:    difficulty,
		}

	case "word_bank":
		// Build the translation word by word
		return exerciseData{
			Type:          "word_bank",
			Prompt:        fmt.Sprintf("Build the translation for: %s", word.Word),
			CorrectAnswer: word.Translation,
			Choices:       generateWordBankChoices(word.Translation),
			Hints:         []string{word.Pinyin},
			Difficulty:    difficulty,
		}

	case "listening":
		// Listen and select
		choices := []string{word.Word}
		for _, v := range allVocab {
			if v.Word != word.Word && len(choices) < 4 {
				choices = append(choices, v.Word)
			}
		}
		return exerciseData{
			Type:          "listening",
			Prompt:        "Select what you hear",
			CorrectAnswer: word.Word,
			Choices:       shuffleStrings(choices),
			AudioURL:      fmt.Sprintf("/audio/zh/%s.mp3", word.Pinyin),
			Hints:         []string{"Listen carefully to the tones"},
			Difficulty:    difficulty,
		}

	case "fill_blank":
		return exerciseData{
			Type:          "fill_blank",
			Prompt:        fmt.Sprintf("___ means \"%s\" in Chinese", word.Translation),
			CorrectAnswer: word.Word,
			Choices:       []string{word.Word},
			Hints:         []string{word.Pinyin},
			Difficulty:    difficulty,
		}

	case "match_pairs":
		pairs := []string{word.Word, word.Translation}
		for _, v := range allVocab {
			if v.Word != word.Word && len(pairs) < 8 {
				pairs = append(pairs, v.Word, v.Translation)
			}
		}
		return exerciseData{
			Type:          "match_pairs",
			Prompt:        "Match the Chinese words with their meanings",
			CorrectAnswer: word.Word,
			Choices:       pairs,
			Hints:         []string{},
			Difficulty:    difficulty,
		}

	default:
		return exerciseData{
			Type:          "multiple_choice",
			Prompt:        fmt.Sprintf("What does \"%s\" mean?", word.Word),
			CorrectAnswer: word.Translation,
			Choices:       []string{word.Translation, "option2", "option3", "option4"},
			Difficulty:    difficulty,
		}
	}
}

func shuffleStrings(s []string) []string {
	// Simple shuffle - in production use proper randomization
	result := make([]string, len(s))
	copy(result, s)
	for i := len(result) - 1; i > 0; i-- {
		j := i / 2 // Deterministic for testing
		result[i], result[j] = result[j], result[i]
	}
	return result
}

func generateWordBankChoices(translation string) []string {
	words := []string{}
	// Split translation into words and add distractors
	word := translation
	words = append(words, word)
	// Add some distractor words
	distractors := []string{"the", "a", "is", "are", "not", "very", "to"}
	for i := 0; i < 3 && i < len(distractors); i++ {
		words = append(words, distractors[i])
	}
	return words
}

func getChineseUnits() []ChineseUnit {
	return []ChineseUnit{
		{
			Title:       "Basics 1",
			Description: "Learn basic greetings and introductions",
			Skills: []ChineseSkill{
				{
					Name:     "Greetings",
					IconName: "wave",
					Vocab: []ChineseVocab{
						{Word: "你好", Pinyin: "nǐ hǎo", Translation: "hello", POS: "interjection"},
						{Word: "早上好", Pinyin: "zǎoshang hǎo", Translation: "good morning", POS: "phrase"},
						{Word: "晚上好", Pinyin: "wǎnshang hǎo", Translation: "good evening", POS: "phrase"},
						{Word: "再见", Pinyin: "zàijiàn", Translation: "goodbye", POS: "interjection"},
						{Word: "谢谢", Pinyin: "xièxie", Translation: "thank you", POS: "interjection"},
					},
				},
				{
					Name:     "Introduction",
					IconName: "user",
					Vocab: []ChineseVocab{
						{Word: "我", Pinyin: "wǒ", Translation: "I", POS: "pronoun"},
						{Word: "你", Pinyin: "nǐ", Translation: "you", POS: "pronoun"},
						{Word: "是", Pinyin: "shì", Translation: "am/is/are", POS: "verb"},
						{Word: "叫", Pinyin: "jiào", Translation: "to be called", POS: "verb"},
						{Word: "什么", Pinyin: "shénme", Translation: "what", POS: "pronoun"},
					},
				},
				{
					Name:     "Common Phrases",
					IconName: "chat",
					Vocab: []ChineseVocab{
						{Word: "请", Pinyin: "qǐng", Translation: "please", POS: "adverb"},
						{Word: "对不起", Pinyin: "duìbuqǐ", Translation: "sorry", POS: "phrase"},
						{Word: "没关系", Pinyin: "méi guānxi", Translation: "no problem", POS: "phrase"},
						{Word: "不客气", Pinyin: "bú kèqi", Translation: "you're welcome", POS: "phrase"},
						{Word: "好", Pinyin: "hǎo", Translation: "good", POS: "adjective"},
					},
				},
			},
		},
		{
			Title:       "Food and Drinks",
			Description: "Name food and drinks",
			Skills: []ChineseSkill{
				{
					Name:     "Drinks",
					IconName: "cup",
					Vocab: []ChineseVocab{
						{Word: "茶", Pinyin: "chá", Translation: "tea", POS: "noun"},
						{Word: "咖啡", Pinyin: "kāfēi", Translation: "coffee", POS: "noun"},
						{Word: "水", Pinyin: "shuǐ", Translation: "water", POS: "noun"},
						{Word: "牛奶", Pinyin: "niúnǎi", Translation: "milk", POS: "noun"},
						{Word: "果汁", Pinyin: "guǒzhī", Translation: "juice", POS: "noun"},
					},
				},
				{
					Name:     "Fruits",
					IconName: "apple",
					Vocab: []ChineseVocab{
						{Word: "苹果", Pinyin: "píngguǒ", Translation: "apple", POS: "noun"},
						{Word: "香蕉", Pinyin: "xiāngjiāo", Translation: "banana", POS: "noun"},
						{Word: "橙子", Pinyin: "chéngzi", Translation: "orange", POS: "noun"},
						{Word: "葡萄", Pinyin: "pútao", Translation: "grape", POS: "noun"},
						{Word: "西瓜", Pinyin: "xīguā", Translation: "watermelon", POS: "noun"},
					},
				},
				{
					Name:     "Meals",
					IconName: "food",
					Vocab: []ChineseVocab{
						{Word: "米饭", Pinyin: "mǐfàn", Translation: "rice", POS: "noun"},
						{Word: "面条", Pinyin: "miàntiáo", Translation: "noodles", POS: "noun"},
						{Word: "鸡蛋", Pinyin: "jīdàn", Translation: "egg", POS: "noun"},
						{Word: "肉", Pinyin: "ròu", Translation: "meat", POS: "noun"},
						{Word: "鱼", Pinyin: "yú", Translation: "fish", POS: "noun"},
					},
				},
			},
		},
		{
			Title:       "Family",
			Description: "Talk about your family",
			Skills: []ChineseSkill{
				{
					Name:     "Parents",
					IconName: "family",
					Vocab: []ChineseVocab{
						{Word: "妈妈", Pinyin: "māma", Translation: "mother", POS: "noun"},
						{Word: "爸爸", Pinyin: "bàba", Translation: "father", POS: "noun"},
						{Word: "父母", Pinyin: "fùmǔ", Translation: "parents", POS: "noun"},
						{Word: "家", Pinyin: "jiā", Translation: "home/family", POS: "noun"},
						{Word: "家人", Pinyin: "jiārén", Translation: "family members", POS: "noun"},
					},
				},
				{
					Name:     "Siblings",
					IconName: "users",
					Vocab: []ChineseVocab{
						{Word: "哥哥", Pinyin: "gēge", Translation: "older brother", POS: "noun"},
						{Word: "弟弟", Pinyin: "dìdi", Translation: "younger brother", POS: "noun"},
						{Word: "姐姐", Pinyin: "jiějie", Translation: "older sister", POS: "noun"},
						{Word: "妹妹", Pinyin: "mèimei", Translation: "younger sister", POS: "noun"},
						{Word: "孩子", Pinyin: "háizi", Translation: "child", POS: "noun"},
					},
				},
				{
					Name:     "Extended Family",
					IconName: "tree",
					Vocab: []ChineseVocab{
						{Word: "爷爷", Pinyin: "yéye", Translation: "grandfather (paternal)", POS: "noun"},
						{Word: "奶奶", Pinyin: "nǎinai", Translation: "grandmother (paternal)", POS: "noun"},
						{Word: "外公", Pinyin: "wàigōng", Translation: "grandfather (maternal)", POS: "noun"},
						{Word: "外婆", Pinyin: "wàipó", Translation: "grandmother (maternal)", POS: "noun"},
						{Word: "亲戚", Pinyin: "qīnqi", Translation: "relatives", POS: "noun"},
					},
				},
			},
		},
		{
			Title:       "Numbers",
			Description: "Learn to count in Chinese",
			Skills: []ChineseSkill{
				{
					Name:     "Numbers 1-10",
					IconName: "number",
					Vocab: []ChineseVocab{
						{Word: "一", Pinyin: "yī", Translation: "one", POS: "number"},
						{Word: "二", Pinyin: "èr", Translation: "two", POS: "number"},
						{Word: "三", Pinyin: "sān", Translation: "three", POS: "number"},
						{Word: "四", Pinyin: "sì", Translation: "four", POS: "number"},
						{Word: "五", Pinyin: "wǔ", Translation: "five", POS: "number"},
						{Word: "六", Pinyin: "liù", Translation: "six", POS: "number"},
						{Word: "七", Pinyin: "qī", Translation: "seven", POS: "number"},
						{Word: "八", Pinyin: "bā", Translation: "eight", POS: "number"},
						{Word: "九", Pinyin: "jiǔ", Translation: "nine", POS: "number"},
						{Word: "十", Pinyin: "shí", Translation: "ten", POS: "number"},
					},
				},
				{
					Name:     "Bigger Numbers",
					IconName: "calculator",
					Vocab: []ChineseVocab{
						{Word: "百", Pinyin: "bǎi", Translation: "hundred", POS: "number"},
						{Word: "千", Pinyin: "qiān", Translation: "thousand", POS: "number"},
						{Word: "万", Pinyin: "wàn", Translation: "ten thousand", POS: "number"},
						{Word: "零", Pinyin: "líng", Translation: "zero", POS: "number"},
						{Word: "多少", Pinyin: "duōshao", Translation: "how many", POS: "pronoun"},
					},
				},
			},
		},
		{
			Title:       "Colors",
			Description: "Learn color words",
			Skills: []ChineseSkill{
				{
					Name:     "Basic Colors",
					IconName: "palette",
					Vocab: []ChineseVocab{
						{Word: "红色", Pinyin: "hóngsè", Translation: "red", POS: "noun"},
						{Word: "蓝色", Pinyin: "lánsè", Translation: "blue", POS: "noun"},
						{Word: "绿色", Pinyin: "lǜsè", Translation: "green", POS: "noun"},
						{Word: "黄色", Pinyin: "huángsè", Translation: "yellow", POS: "noun"},
						{Word: "白色", Pinyin: "báisè", Translation: "white", POS: "noun"},
						{Word: "黑色", Pinyin: "hēisè", Translation: "black", POS: "noun"},
					},
				},
				{
					Name:     "More Colors",
					IconName: "rainbow",
					Vocab: []ChineseVocab{
						{Word: "橙色", Pinyin: "chéngsè", Translation: "orange", POS: "noun"},
						{Word: "紫色", Pinyin: "zǐsè", Translation: "purple", POS: "noun"},
						{Word: "粉色", Pinyin: "fěnsè", Translation: "pink", POS: "noun"},
						{Word: "灰色", Pinyin: "huīsè", Translation: "gray", POS: "noun"},
						{Word: "棕色", Pinyin: "zōngsè", Translation: "brown", POS: "noun"},
					},
				},
			},
		},
		{
			Title:       "Time",
			Description: "Tell the time and talk about days",
			Skills: []ChineseSkill{
				{
					Name:     "Days of Week",
					IconName: "calendar",
					Vocab: []ChineseVocab{
						{Word: "星期一", Pinyin: "xīngqīyī", Translation: "Monday", POS: "noun"},
						{Word: "星期二", Pinyin: "xīngqī'èr", Translation: "Tuesday", POS: "noun"},
						{Word: "星期三", Pinyin: "xīngqīsān", Translation: "Wednesday", POS: "noun"},
						{Word: "星期四", Pinyin: "xīngqīsì", Translation: "Thursday", POS: "noun"},
						{Word: "星期五", Pinyin: "xīngqīwǔ", Translation: "Friday", POS: "noun"},
						{Word: "星期六", Pinyin: "xīngqīliù", Translation: "Saturday", POS: "noun"},
						{Word: "星期天", Pinyin: "xīngqītiān", Translation: "Sunday", POS: "noun"},
					},
				},
				{
					Name:     "Time Words",
					IconName: "clock",
					Vocab: []ChineseVocab{
						{Word: "今天", Pinyin: "jīntiān", Translation: "today", POS: "noun"},
						{Word: "明天", Pinyin: "míngtiān", Translation: "tomorrow", POS: "noun"},
						{Word: "昨天", Pinyin: "zuótiān", Translation: "yesterday", POS: "noun"},
						{Word: "现在", Pinyin: "xiànzài", Translation: "now", POS: "adverb"},
						{Word: "小时", Pinyin: "xiǎoshí", Translation: "hour", POS: "noun"},
					},
				},
			},
		},
		{
			Title:       "Places",
			Description: "Learn about locations",
			Skills: []ChineseSkill{
				{
					Name:     "Common Places",
					IconName: "building",
					Vocab: []ChineseVocab{
						{Word: "学校", Pinyin: "xuéxiào", Translation: "school", POS: "noun"},
						{Word: "医院", Pinyin: "yīyuàn", Translation: "hospital", POS: "noun"},
						{Word: "商店", Pinyin: "shāngdiàn", Translation: "store", POS: "noun"},
						{Word: "餐厅", Pinyin: "cāntīng", Translation: "restaurant", POS: "noun"},
						{Word: "银行", Pinyin: "yínháng", Translation: "bank", POS: "noun"},
					},
				},
				{
					Name:     "Directions",
					IconName: "compass",
					Vocab: []ChineseVocab{
						{Word: "左", Pinyin: "zuǒ", Translation: "left", POS: "noun"},
						{Word: "右", Pinyin: "yòu", Translation: "right", POS: "noun"},
						{Word: "前", Pinyin: "qián", Translation: "front", POS: "noun"},
						{Word: "后", Pinyin: "hòu", Translation: "back", POS: "noun"},
						{Word: "这里", Pinyin: "zhèlǐ", Translation: "here", POS: "pronoun"},
						{Word: "那里", Pinyin: "nàlǐ", Translation: "there", POS: "pronoun"},
					},
				},
			},
		},
		{
			Title:       "Activities",
			Description: "Describe daily activities",
			Skills: []ChineseSkill{
				{
					Name:     "Daily Verbs",
					IconName: "run",
					Vocab: []ChineseVocab{
						{Word: "吃", Pinyin: "chī", Translation: "to eat", POS: "verb"},
						{Word: "喝", Pinyin: "hē", Translation: "to drink", POS: "verb"},
						{Word: "睡觉", Pinyin: "shuìjiào", Translation: "to sleep", POS: "verb"},
						{Word: "工作", Pinyin: "gōngzuò", Translation: "to work", POS: "verb"},
						{Word: "学习", Pinyin: "xuéxí", Translation: "to study", POS: "verb"},
					},
				},
				{
					Name:     "More Actions",
					IconName: "activity",
					Vocab: []ChineseVocab{
						{Word: "看", Pinyin: "kàn", Translation: "to look/watch", POS: "verb"},
						{Word: "听", Pinyin: "tīng", Translation: "to listen", POS: "verb"},
						{Word: "说", Pinyin: "shuō", Translation: "to speak", POS: "verb"},
						{Word: "读", Pinyin: "dú", Translation: "to read", POS: "verb"},
						{Word: "写", Pinyin: "xiě", Translation: "to write", POS: "verb"},
					},
				},
			},
		},
		{
			Title:       "Weather",
			Description: "Talk about the weather",
			Skills: []ChineseSkill{
				{
					Name:     "Weather Words",
					IconName: "cloud",
					Vocab: []ChineseVocab{
						{Word: "天气", Pinyin: "tiānqì", Translation: "weather", POS: "noun"},
						{Word: "太阳", Pinyin: "tàiyáng", Translation: "sun", POS: "noun"},
						{Word: "下雨", Pinyin: "xiàyǔ", Translation: "to rain", POS: "verb"},
						{Word: "下雪", Pinyin: "xiàxuě", Translation: "to snow", POS: "verb"},
						{Word: "冷", Pinyin: "lěng", Translation: "cold", POS: "adjective"},
						{Word: "热", Pinyin: "rè", Translation: "hot", POS: "adjective"},
					},
				},
			},
		},
		{
			Title:       "Shopping",
			Description: "Go shopping in Chinese",
			Skills: []ChineseSkill{
				{
					Name:     "Shopping Basics",
					IconName: "cart",
					Vocab: []ChineseVocab{
						{Word: "买", Pinyin: "mǎi", Translation: "to buy", POS: "verb"},
						{Word: "卖", Pinyin: "mài", Translation: "to sell", POS: "verb"},
						{Word: "钱", Pinyin: "qián", Translation: "money", POS: "noun"},
						{Word: "便宜", Pinyin: "piányi", Translation: "cheap", POS: "adjective"},
						{Word: "贵", Pinyin: "guì", Translation: "expensive", POS: "adjective"},
					},
				},
				{
					Name:     "Clothes",
					IconName: "shirt",
					Vocab: []ChineseVocab{
						{Word: "衣服", Pinyin: "yīfu", Translation: "clothes", POS: "noun"},
						{Word: "裤子", Pinyin: "kùzi", Translation: "pants", POS: "noun"},
						{Word: "鞋子", Pinyin: "xiézi", Translation: "shoes", POS: "noun"},
						{Word: "帽子", Pinyin: "màozi", Translation: "hat", POS: "noun"},
						{Word: "大", Pinyin: "dà", Translation: "big", POS: "adjective"},
						{Word: "小", Pinyin: "xiǎo", Translation: "small", POS: "adjective"},
					},
				},
			},
		},
	}
}
