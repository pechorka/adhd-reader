package bot

const (
	helpMsg = `Hello!
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
)
