package main

// Caption, window title, greeting, request, footer.
var sponsorTranslations = map[string][5]string{
	"zh-CN": {"赞助作者一杯咖啡", "感谢支持 · 赞助作者", "感谢使用 OPPO P5 Repair！", "赞助作者一杯咖啡，支持持续完善软件。", "自愿赞助，感谢每一份支持！"},
	"zh-TW": {"贊助作者一杯咖啡", "感謝支持 · 贊助作者", "感謝使用 OPPO P5 Repair！", "贊助作者一杯咖啡，支持持續改善軟體。", "自由贊助，感謝每一份支持！"},
	"en":    {"Buy the author a coffee", "Thank you for your support", "Thank you for using OPPO P5 Repair!", "Buy the author a coffee and support continued development.", "Donations are optional. Every contribution is appreciated!"},
	"ja":    {"作者にコーヒーを贈る", "ご支援ありがとうございます", "OPPO P5 Repair をご利用いただきありがとうございます！", "作者へのコーヒー代のご支援が、ソフトの改善につながります。", "ご支援は任意です。温かい応援に感謝します！"},
	"ko":    {"개발자에게 커피 후원", "후원해 주셔서 감사합니다", "OPPO P5 Repair를 이용해 주셔서 감사합니다!", "개발자에게 커피 한 잔을 후원하고 지속적인 개선을 응원해 주세요.", "후원은 자유입니다. 모든 응원에 감사드립니다!"},
	"fr":    {"Offrir un café à l’auteur", "Merci pour votre soutien", "Merci d’utiliser OPPO P5 Repair !", "Offrez un café à l’auteur pour soutenir le développement.", "Le don est facultatif. Merci pour chaque contribution !"},
	"de":    {"Dem Autor Kaffee spendieren", "Vielen Dank für Ihre Unterstützung", "Vielen Dank, dass Sie OPPO P5 Repair nutzen!", "Spendieren Sie dem Autor einen Kaffee und fördern Sie die Entwicklung.", "Spenden sind freiwillig. Vielen Dank für jeden Beitrag!"},
	"it":    {"Offri un caffè all’autore", "Grazie per il sostegno", "Grazie per aver scelto OPPO P5 Repair!", "Offri un caffè all’autore per sostenere lo sviluppo del software.", "La donazione è facoltativa. Grazie per ogni contributo!"},
	"es":    {"Invita al autor a un café", "Gracias por tu apoyo", "¡Gracias por usar OPPO P5 Repair!", "Invita al autor a un café y apoya el desarrollo del programa.", "La donación es opcional. ¡Gracias por cada aportación!"},
	"pt":    {"Ofereça um café ao autor", "Obrigado pelo seu apoio", "Obrigado por utilizar o OPPO P5 Repair!", "Ofereça um café ao autor e apoie o desenvolvimento do programa.", "O donativo é opcional. Obrigado por cada contribuição!"},
	"ru":    {"Угостить автора кофе", "Спасибо за вашу поддержку", "Спасибо за использование OPPO P5 Repair!", "Угостите автора кофе и поддержите развитие программы.", "Пожертвования добровольны. Спасибо за любую поддержку!"},
	"ar":    {"قدّم قهوة للمطوّر", "شكرًا لدعمك", "شكرًا لاستخدام OPPO P5 Repair!", "قدّم قهوة للمطوّر وساهم في مواصلة تطوير البرنامج.", "التبرع اختياري. شكرًا لكل مساهمة!"},
}

func sponsorLocalized(language string, index int) string {
	row, ok := sponsorTranslations[language]
	if !ok {
		row = sponsorTranslations["en"]
	}
	return row[index]
}
func sponsorMessage(index int) string { return sponsorLocalized(app.language, index) }

func refreshSponsorLanguage() {
	setText(app.labels["sponsorCaption"], sponsorMessage(0))
	pInvalidate.Call(app.labels["sponsorCaption"], 0, 1)
	if supportWindow != 0 {
		setText(supportWindow, sponsorMessage(1))
		pInvalidate.Call(supportWindow, 0, 0)
	}
}
