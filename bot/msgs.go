package bot

import _ "embed"

//go:embed initialFiles/Your.attention.span.is.shrinking.txt
var startFile []byte

const startFileName = "Your.attention.span.is.shrinking.txt"

// onboarding messages

const (
	firstMsg = `Welcome to ADHD Reading Bot! 📚📖
	We live busy lives.
	It's hard to find time to read books or articles or even posts in telegram channels... But it's easy to find this 1 minute to look at this cutie cat picture in Telegram 😍`
	secondMsg = `This bot can help you chunk books, articles, or long-read posts into smaller segments.
	1️⃣ Easy to digest. 🤤
	Choose your own size of segments. The default is only 500 symbols. (1 short paragraph)
	2️⃣ Easy to start reading. 🚀
	Right in Telegram, next to cute kitties.
	3️⃣ Easy to stop reading 🛑
	No more remembering which paragraph you stopped at
	4️⃣ Easy to share! 🤝
	No more excruciating selecting of words, just Forward whole chunk to your Telegram contacts or a group`
	thirdMsg  = `👀🧩Choose your chunk size! The default is 500. And you can always change it using /chunk command, it will apply to all new texts. Take a look at different chunk sizes from "Your attention span is shrinking, studies say. Here’s how to stay focused" by CNN.`
	fourthMsg = `📝 This is 250 symbols chunk
	“In 2004, we measured the average attention on a screen to be 2½ minutes,” Mark said. “Some years later, we found attention spans to be about 75 seconds. Now we find people can only pay attention to one screen for an average of 47 seconds.”`
	fifthMsg = `📝 This is 500 symbols chunk
	“With the exception of a few rare individuals, there is no such thing as multitasking,” Mark said. “Unless one of the tasks is automatic, like chewing gum or walking, you cannot do two effortful things at the same time.
	For example, she said, you can’t read email and be in a video meeting. When you focus on one, you lose the other. “You’re actually switching your attention very quickly between the two. And when you switch your attention fast, it’s correlated with stress,” Mark explained.`
	sixthMsg = `📚📝To get started, send a text file (for now it's only .txt) or message to this chat, and then press the button "Read" to start reading the first segment! If you don't have text at hand to start, here is the file to start. Forward it to the bot to add to your library.`
	// seventhMsg is a file
	eighthMsg = `📋👀 Use command /list to get a list of your texts. Choose one to read now!
🔢 Use command /page [integer number] to quickly go to a specific chunk. For example, /page 2
❌Use command /delete [name of the text] to delete text from the library. For example, /delete Your.attention.span.is.shrinking.txt

🆘 If you have any questions or need help, try out /help command or just send a message to @rubella19 and we'll get back to you as soon as possible.`
)

const helpMsg = `Hello!
	Let's review bot commands:
	📋 Use command /list to get a list of your texts.
	🔢 Use command /page [integer number] to quickly go to a specific chunk. It works after you selected text using command /list or pressed the button "Read" after text uploading. Example, /page 2
	❌Use command /delete [name of the text] to delete text from the library. You can copy text name from the message from the bot when selecting text from the list. For example, /delete Your.attention.span.is.shrinking.txt
	🧩 Use command /chunk [integer number] to set your preferred chunk size. It takes numbers from 1 to 4096. The default is 500. It's the size of a small paragraph. Typically 2 chunks of this size fit on the mobile phone screen.
	
	🌟Features, not bugs
	Only accepts UTF-8 encoding.
	For now, accepts only .txt files no bigger than ~20Mb
	
	🐞 Bugs with low priority and an unclear way to solve... so let's live with them
	When forwarding messages "Prev\Next" buttons disappear...
	Not perfect handling for citations chunking
	Pictures handling
	
	If you encountered a bug or odd behavior you can write to 👩🏻‍🦰 @rubella19 or 🎁 simply create an <a href="https://github.com/pechorka/adhd-reader/issues">issue on GitHub</a>
	
	🆘 If you have any further questions or need help just send a message to @rubella19 and we'll get back to you as soon as possible.`

const textDeletedMsg = "Text deleted. Let's choose something to read: /list"
