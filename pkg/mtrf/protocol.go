package mtrf

// https://www.noo.com.by/assets/files/PDF/MTRF-64.pdf

// MessageLen длина отправляемых и принимаемых сообщений
const MessageLen = 17

const (
	// ModeTX - режим nooLite TX
	ModeTX uint8 = 0
	// ModeRX - режим nooLite RX
	ModeRX = 1
	// ModeTXF - режим nooLite-F TX
	ModeTXF = 2
	// ModeRXF - режим nooLite-F RX
	ModeRXF = 3
	// ModeService - сервисный режим работы с nooLite-F
	ModeService = 4
	// ModeUpgrade - режим обновления ПО nooLite-F
	ModeUpgrade = 5
)

const (
	// CmdOff - Выключить нагрузку
	CmdOff uint8 = 0
	// CmdBrightDown - Запускает плавное понижение яркости
	CmdBrightDown = 1
	// CmdOn - Включить нагрузку
	CmdOn = 2
	// CmdBrightUp - Запускает плавное повышение яркости вниз
	CmdBrightUp = 3
	// CmdSwitch - Включает или выключает нагрузку
	CmdSwitch = 4
	// CmdBrightBack - Запускает плавное изменение яркости в обратном направлении
	CmdBrightBack = 5
	// CmdSetBrightness - Установить заданную в расширении команды яркость (количество данных зависит от устройства)
	CmdSetBrightness = 6
	// CmdLoadPreset - Вызвать записанный сценарий
	CmdLoadPreset = 7
	// CmdSavePreset - Записать сценарий в память
	CmdSavePreset = 8
	// CmdUnbind - Запускает процедуру стирания адреса управляющего устройства из памяти исполнительного
	CmdUnbind = 9
	// CmdStopReg - Прекращает действие команд Bright_Down, Bright_Up, Bright_Back
	CmdStopReg = 10
	// CmdBrightStepDown - Понизить яркость на шаг. При отсутствии поля данных увеличивает отсечку на 64 мкс, при наличии поля данных на величину в микросекундах (0 соответствует 256 мкс)
	CmdBrightStepDown = 11
	// CmdBrightStepUp - Повысить яркость на шаг. При отсутствии поля данных уменьшает отсечку на 64 мкс, при наличии поля данных на величину в микросекундах (0 соответствует 256 мкс)
	CmdBrightStepUp = 12
	// CmdBrightReg - Запускает плавное изменение яркости с направлением и скоростью, заданными в расширении
	CmdBrightReg = 13
	// CmdBind - Сообщает исполнительному устройству, что управляющее хочет активировать режим привязки. При привязке также передается тип устройства в данных
	CmdBind = 15
	// CmdRollColour - Запускает плавное изменение цвета в RGBконтроллере по радуге
	CmdRollColour = 16
	// CmdSwitchColour - Переключение между стандартными цветами в RGB-контроллере
	CmdSwitchColour = 17
	// CmdSwitchMode - Переключение между режимами RGBконтроллера
	CmdSwitchMode = 18
	// CmdSpeedModeBack - Запускает изменение скорости работы режимов RGB контроллера в обратном направлении
	CmdSpeedModeBack = 19
	// CmdBatteryLow - У устройства, которое передало данную команду, разрядился элемент питания
	CmdBatteryLow = 20
	// CmdSensTempHumi - Передает данные о температуре, влажности и состоянии элементов
	CmdSensTempHumi = 21
	// CmdTemporaryOn - Включить свет на заданное время. Время в 5-секундных тактах передается в расширении (см.описание A)
	CmdTemporaryOn = 25
	// CmdModes - Установка режимов работы исполнительного устройства (см. описание B)
	CmdModes = 26
	// CmdReadState - Получение состояния исполнительного устройства (см. описание C)
	CmdReadState = 128
	// CmdWriteState - Установка состояния исполнительного устройства
	CmdWriteState = 129
	// CmdSendState - Ответ от исполнительного устройства (см. описание C)
	CmdSendState = 130
	// CmdService - Включение сервисного режима на заранее привязанном устройстве (см. описание D)
	CmdService = 131
	// CmdClearMemory - Очистка памяти устройства nooLite. Для выполнения команды используется ключ 170-85-170-85 (записывается в поле данных D0…D3)
	CmdClearMemory = 132
)

// Request - запросы, отправляемые модулю MTRF-64
type Request struct {
	// Стартовый байт
	// Значение=171
	St uint8
	// Режим работы модуля
	// Значение=0 – режим nooLite TX
	// Значение=1 – режим nooLite RX
	// Значение=2 – режим nooLite-F TX
	// Значение=3 – режим nooLite-F RX
	// Значение=4 – сервисный режим работы с nooLite-F
	// Значение=5 – режим обновления ПО nooLite-F
	Mode uint8
	// Управление модулем
	// Бит 5…0 – Команда модулю (0…63)
	// Значение=0 – Передать команду
	// Значение=1 – Передать широковещательную команду (одновременно всем устройствам на канале CH)
	// Значение=2 – Считать ответ (состояние приемного буфера)
	// Значение=3 – Включить привязку
	// Значение=4 – Выключить привязку
	// Значение=5 – Очистить ячейку (канал)
	// Значение=6 – Очистить память (все каналы)
	// Значение=7 – Отвязать адрес от канала
	// Значение=8 – Передать команду по указанному адресу nooLite-F
	// Бит 6…7 – Nrep, количество дополнительных повторов команды (0...3). Количество передач команд = 2+Nrep
	Ctr uint8
	// Зарезервирован, не используется
	// Значение=0
	Res uint8
	// Адрес канала, ячейки привязки
	// Значение адреса канала или номера ячейки для привязки: 0…63 для MTRF-64
	Ch uint8
	// Команда, отправляемая с модуля. См. описание в таблице «Список команд»
	Cmd uint8
	// Формат
	// Количество данных, передаваемых вместе с командой и их назначение. См. описание команд в таблице «Список команд»
	Fmt uint8
	// Байт данных 0 См. описание в таблице «Формат и Данные»
	D0 uint8
	// Байт данных 1 См. описание в таблице «Формат и Данные»
	D1 uint8
	// Байт данных 2 См. описание в таблице «Формат и Данные»
	D2 uint8
	// Байт данных 3 См. описание в таблице «Формат и Данные»
	D3 uint8
	// Идентификатор блока, бит 31…24
	// Адрес устройства в системе nooLite-F, которому предназначается команда
	ID0 uint8
	// Идентификатор блока, бит 23…16
	// Адрес устройства в системе nooLite-F
	ID1 uint8
	// Идентификатор блока, бит 15…8
	// Адрес устройства в системе nooLite-F
	ID2 uint8
	// Идентификатор блока, бит 7…0
	// Адрес устройства в системе nooLite-F
	ID3 uint8
	// Контрольная сумма
	// Младший байт от суммы первых 15 байт (ST… ID3).
	Crc uint8
	// Стоповый байт
	// Значение=172
	Sp uint8
}

// Response -  данные, получаемые с модуля MTRF-64 (считываемые или отправляемые автоматически после передачи команд с выдачей ответа)
type Response struct {
	// Стартовый байт
	// Значение=173
	St uint8
	// Режим работы модуля
	// Значение=0 – режим nooLite TX
	// Значение=1 – режим nooLite RX
	// Значение=2 – режим nooLite-F TX
	// Значение=3 – режим nooLite-F RX
	// Значение=4 – сервисный режим работы с nooLite-F
	// Значение=5 – режим обновления ПО nooLite-F
	Mode uint8
	// Код ответа
	// Команда модулю:
	// Значение=0 – Команда выполнена
	// Значение=1 – Нет ответа от блока
	// Значение=2 – Ошибка во время выполнения
	// Значение=3 – Привязка выполнена
	Ctr uint8
	// Количество оставшихся ответов от модуля, значение TOGL
	// Для nooLite-F TX:
	// В значении приводится количество пакетов, которые осталось передать модулю для завершения опроса канала.
	// Для nooLite RX и nooLite-F RX:
	// Значение TOGL. Изменяется при приходе новой команды на модуль (увеличивается на единицу).
	Togl uint8
	// Адрес канала, ячейки привязки
	// Значение адреса канала или номера ячейки для которого была принята команда: 0…63 для MTRF-64
	Ch uint8
	// Команда, принимаемая модулем. См. описание в таблице «Список команд»
	Cmd uint8
	// Формат
	// Количество данных, передаваемых вместе с командой и их назначение. См. описание в таблице «Формат и Данные»
	Fmt uint8
	// Байт данных 0 См. описание в таблице «Формат и Данные»
	D0 uint8
	// Байт данных 1 См. описание в таблице «Формат и Данные»
	D1 uint8
	// Байт данных 2 См. описание в таблице «Формат и Данные»
	D2 uint8
	// Байт данных 3 См. описание в таблице «Формат и Данные»
	D3 uint8
	// Идентификатор блока, бит 31…24
	// Адрес устройства (32 бита) в системе nooLite-F, которое передало команду
	ID0 uint8
	// Идентификатор блока, бит 23…16
	// Адрес устройства (32 бита) в системе nooLite-F, которое передало команду
	ID1 uint8
	// Идентификатор блока, бит 15…8
	// Адрес устройства (32 бита) в системе nooLite-F, которое передало команду
	ID2 uint8
	// Идентификатор блока, бит 7…0
	// Адрес устройства (32 бита) в системе nooLite-F, которое передало команду
	ID3 uint8
	// Контрольная сумма
	// Младший байт от суммы первых 15 байт (ST… ID3).
	Crc uint8
	// Стоповый байт
	// Значение=174
	Sp uint8
}
