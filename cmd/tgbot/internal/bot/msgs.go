package bot

import _ "embed"

//go:embed initialFiles/Your.attention.span.is.shrinking.txt
var startFileEn []byte

//go:embed initialFiles/Обучение_в_эпоху_золотых_рыбок.txt
var startFileRu []byte

const startFileNameEn = "Your attention span is shrinking.txt"
const startFileNameRu = `Обучение в эпоху "золотых_рыбок".txt`

// error messages
const (
	panicMsgId                            = "panic"
	errorOnTextSelectMsgId                = "error_on_text_select"
	errorOnTextDeleteMsgId                = "error_on_text_delete"
	errorOnTextDeleteExampleTextMsgId     = "error_on_text_delete_example_text"
	errorOnTextDeleteNoTextsAddedMsgId    = "error_on_text_delete_no_texts_added"
	erroroOnGettingNextChunk              = "error_on_getting_next_chunk"
	errorOnListMsgId                      = "error_on_list"
	errorOnParsingPageMsgId               = "error_on_parsing_page"
	errorOnSettingPageNoTextSelectedMsgId = "error_on_setting_page_no_text_selected"
	errorOnSettingPageMsgId               = "error_on_setting_page"
	errorOnParsingChunkSizeMsgId          = "error_on_parsing_chunk_size"
	errorOnSettingChunkSizeMsgId          = "error_on_setting_chunk_size"
	errorOnFileUploadTooBigMsgId          = "error_on_file_upload_too_big"
	errorOnFileUploadInvalidFormatMsgId   = "error_on_file_upload_invalid_format"
	errorOnFileUploadBuildingFileURLMsgId = "error_on_file_upload_building_file_URL"
	errorOnFileUploadExtractingTextMsgId  = "error_on_file_upload_extracting_text"
	errorOnFileUploadMsgId                = "error_on_file_upload"
	errorOnTextSaveNotUTF8MsgId           = "error_on_text_save_not_utf8"
	errorOnTextSaveMsgId                  = "error_on_text_save"
	errorUnknownCommandMsgId              = "error_unknown_command"
)

const (
	onTextSelectMsgId  = "on_text_select"
	onTextDeletedMsgId = "on_text_deleted"
	textFinishedMsgId  = "text_finished"
	lastChunkMsgId     = "last_chunk"
	onListMsgId        = "on_list"
	pageSetMsgId       = "page_set"
	chunkSizeSetMsgId  = "chunk_size_set"
	textSavedMsgId     = "text_saved"
)

const (
	previousButtonMsgId           = "previous_button"
	nextButtonMsgId               = "next_button"
	deleteButtonMsgId             = "delete_button"
	deleteButtonWithTextNameMsgId = "delete_button_with_text_name"
	readButtonMsgId               = "read_button"
	rereadButtonMsgId             = "reread_button"
)

const (
	warningFirstChunkCantGoBackMsgId = "warning_first_chunk_cant_go_back"
	warningNoTextsMsgId              = "warning_no_texts"
)

// onboarding messages

const (
	firstMsg  = "onboarding_first_msg"
	secondMsg = "onboarding_second_msg"
	thirdMsg  = "onboarding_third_msg"
	fourthMsg = "onboarding_fourth_msg"
	fifthMsg  = "onboarding_fifth_msg"
	sixthMsg  = "onboarding_sixth_msg"
	// seventhMsg is a file
	eighthMsg = "onboarding_eighth_msg"
)

const helpMsg = "help_msg"
