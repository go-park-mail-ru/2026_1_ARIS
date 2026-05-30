WITH public_seed(position, question_text_ru, question_text_en, correct_answer) AS (
    VALUES
        (1, 'Сколько лет длилась Столетняя война?', 'How many years did the Hundred Years'' War last?', 116::DOUBLE PRECISION),
        (2, 'Сколько раз часовая и минутная стрелки часов совпадают за сутки?', 'How many times do the hour and minute hands of a clock coincide in one day?', 22::DOUBLE PRECISION),
        (3, 'Сколько бит содержит SHA-256?', 'How many bits does SHA-256 contain?', 256::DOUBLE PRECISION),
        (4, 'Сколько животных каждого вида взял Моисей на ковчег?', 'How many animals of each kind did Moses take onto the ark?', 0::DOUBLE PRECISION),
        (5, 'В каком году состоялся первый набор в Технопарк Mail.ru (ныне VK Education) ?', 'In what year was the first intake for Technopark Mail.ru (now VK Education) held?', 2011::DOUBLE PRECISION)
),
updated_existing AS (
    UPDATE game_question AS q
    SET question_text = s.question_text_ru,
        question_text_ru = s.question_text_ru,
        question_text_en = s.question_text_en,
        correct_answer = s.correct_answer,
        is_active = TRUE,
        updated_at = NOW()
    FROM public_seed AS s
    JOIN game_public_lobby_question AS plq ON plq.position = s.position
    WHERE q.id = plq.question_id
    RETURNING s.position, q.id AS question_id
),
inserted_missing AS (
    INSERT INTO game_question (uid, game_type, question_text, question_text_ru, question_text_en, correct_answer, is_active)
    SELECT gen_random_uuid(), 'number_duel', s.question_text_ru, s.question_text_ru, s.question_text_en, s.correct_answer, TRUE
    FROM public_seed AS s
    WHERE NOT EXISTS (
        SELECT 1
        FROM updated_existing AS u
        WHERE u.position = s.position
    )
    RETURNING id, question_text_ru
),
resolved AS (
    SELECT position, question_id
    FROM updated_existing
    UNION ALL
    SELECT s.position, i.id
    FROM public_seed AS s
    JOIN inserted_missing AS i ON i.question_text_ru = s.question_text_ru
)
INSERT INTO game_public_lobby_question (position, question_id)
SELECT position, question_id
FROM resolved
ON CONFLICT (position) DO UPDATE
SET question_id = EXCLUDED.question_id,
    updated_at = NOW();

DELETE FROM game_public_lobby_question
WHERE position > 5;

UPDATE game_room
SET question_count = 5,
    updated_at = NOW()
WHERE is_public_lobby = TRUE
  AND status = 'waiting';

WITH ordinary_seed(position, question_text_ru, question_text_en, correct_answer) AS (
    VALUES
        (1, 'Сколько секунд в одной минуте?', 'How many seconds are in one minute?', 60::DOUBLE PRECISION),
        (2, 'Сколько минут в одном часе?', 'How many minutes are in one hour?', 60::DOUBLE PRECISION),
        (3, 'Сколько часов в сутках?', 'How many hours are in a day?', 24::DOUBLE PRECISION),
        (4, 'Сколько дней в неделе?', 'How many days are in a week?', 7::DOUBLE PRECISION),
        (5, 'Сколько месяцев в году?', 'How many months are in a year?', 12::DOUBLE PRECISION),
        (6, 'Сколько дней в високосном году?', 'How many days are in a leap year?', 366::DOUBLE PRECISION),
        (7, 'Сколько градусов в прямом угле?', 'How many degrees are in a right angle?', 90::DOUBLE PRECISION),
        (8, 'Сколько сторон у треугольника?', 'How many sides does a triangle have?', 3::DOUBLE PRECISION),
        (9, 'Сколько сторон у шестиугольника?', 'How many sides does a hexagon have?', 6::DOUBLE PRECISION),
        (10, 'Сколько сторон у восьмиугольника?', 'How many sides does an octagon have?', 8::DOUBLE PRECISION),
        (11, 'Сколько сантиметров в одном метре?', 'How many centimeters are in one meter?', 100::DOUBLE PRECISION),
        (12, 'Сколько метров в одном километре?', 'How many meters are in one kilometer?', 1000::DOUBLE PRECISION),
        (13, 'Сколько граммов в одном килограмме?', 'How many grams are in one kilogram?', 1000::DOUBLE PRECISION),
        (14, 'Сколько миллиметров в одном сантиметре?', 'How many millimeters are in one centimeter?', 10::DOUBLE PRECISION),
        (15, 'Сколько бит в IPv4-адресе?', 'How many bits are in an IPv4 address?', 32::DOUBLE PRECISION),
        (16, 'Сколько бит в IPv6-адресе?', 'How many bits are in an IPv6 address?', 128::DOUBLE PRECISION),
        (17, 'Сколько уровней в модели OSI?', 'How many layers are in the OSI model?', 7::DOUBLE PRECISION),
        (18, 'Какой максимальный номер TCP-порта?', 'What is the maximum TCP port number?', 65535::DOUBLE PRECISION),
        (19, 'Какой стандартный порт у HTTP?', 'What is the default port for HTTP?', 80::DOUBLE PRECISION),
        (20, 'Какой стандартный порт у HTTPS?', 'What is the default port for HTTPS?', 443::DOUBLE PRECISION),
        (21, 'Какой стандартный порт у DNS?', 'What is the default port for DNS?', 53::DOUBLE PRECISION),
        (22, 'Сколько байтов в UUID?', 'How many bytes are in a UUID?', 16::DOUBLE PRECISION),
        (23, 'Сколько бит в хеше MD5?', 'How many bits are in an MD5 hash?', 128::DOUBLE PRECISION),
        (24, 'Сколько бит в хеше SHA-1?', 'How many bits are in a SHA-1 hash?', 160::DOUBLE PRECISION),
        (25, 'Сколько бит в хеше SHA-512?', 'How many bits are in a SHA-512 hash?', 512::DOUBLE PRECISION),
        (26, 'Какое основание у двоичной системы счисления?', 'What is the base of the binary numeral system?', 2::DOUBLE PRECISION),
        (27, 'Сколько шестнадцатеричных цифр кодируют один байт?', 'How many hexadecimal digits encode one byte?', 2::DOUBLE PRECISION),
        (28, 'Какое максимальное значение у беззнакового байта?', 'What is the maximum value of an unsigned byte?', 255::DOUBLE PRECISION),
        (29, 'Сколько символов в базовой таблице ASCII?', 'How many characters are in the basic ASCII table?', 128::DOUBLE PRECISION),
        (30, 'Сколько букв в английском алфавите?', 'How many letters are in the English alphabet?', 26::DOUBLE PRECISION),
        (31, 'Сколько букв в русском алфавите?', 'How many letters are in the Russian alphabet?', 33::DOUBLE PRECISION),
        (32, 'Какое значение у римской цифры X?', 'What is the value of the Roman numeral X?', 10::DOUBLE PRECISION),
        (33, 'Какое значение у римской цифры L?', 'What is the value of the Roman numeral L?', 50::DOUBLE PRECISION),
        (34, 'Какое значение у римской цифры C?', 'What is the value of the Roman numeral C?', 100::DOUBLE PRECISION),
        (35, 'Какое значение у римской цифры M?', 'What is the value of the Roman numeral M?', 1000::DOUBLE PRECISION),
        (36, 'Сколько клавиш у стандартного фортепиано?', 'How many keys does a standard piano have?', 88::DOUBLE PRECISION),
        (37, 'Сколько белых клавиш у стандартного фортепиано?', 'How many white keys does a standard piano have?', 52::DOUBLE PRECISION),
        (38, 'Сколько чёрных клавиш у стандартного фортепиано?', 'How many black keys does a standard piano have?', 36::DOUBLE PRECISION),
        (39, 'Сколько струн у стандартной гитары?', 'How many strings does a standard guitar have?', 6::DOUBLE PRECISION),
        (40, 'Сколько струн у скрипки?', 'How many strings does a violin have?', 4::DOUBLE PRECISION),
        (41, 'Сколько горизонталей на шахматной доске?', 'How many ranks are on a chessboard?', 8::DOUBLE PRECISION),
        (42, 'Сколько вертикалей на шахматной доске?', 'How many files are on a chessboard?', 8::DOUBLE PRECISION),
        (43, 'Сколько фигур у одной стороны в начале шахматной партии?', 'How many pieces does one side have at the start of a chess game?', 16::DOUBLE PRECISION),
        (44, 'Сколько пешек у одной стороны в начале шахматной партии?', 'How many pawns does one side have at the start of a chess game?', 8::DOUBLE PRECISION),
        (45, 'Сколько фигур всего стоит на доске в начале шахматной партии?', 'How many pieces are on the board at the start of a chess game?', 32::DOUBLE PRECISION),
        (46, 'Сколько карт в стандартной колоде без джокеров?', 'How many cards are in a standard deck without jokers?', 52::DOUBLE PRECISION),
        (47, 'Сколько мастей в стандартной колоде карт?', 'How many suits are in a standard deck of cards?', 4::DOUBLE PRECISION),
        (48, 'Сколько карт одной масти в стандартной колоде?', 'How many cards of one suit are in a standard deck?', 13::DOUBLE PRECISION),
        (49, 'Сколько граней у обычного кубика?', 'How many faces does a standard die have?', 6::DOUBLE PRECISION),
        (50, 'Какова сумма чисел на противоположных гранях обычного кубика?', 'What is the sum of numbers on opposite faces of a standard die?', 7::DOUBLE PRECISION),
        (51, 'Сколько костей в наборе домино double-six?', 'How many tiles are in a double-six domino set?', 28::DOUBLE PRECISION),
        (52, 'Сколько кеглей в боулинге?', 'How many pins are used in bowling?', 10::DOUBLE PRECISION),
        (53, 'Сколько фреймов в партии боулинга?', 'How many frames are in a game of bowling?', 10::DOUBLE PRECISION),
        (54, 'Сколько игроков одной команды на площадке в волейболе?', 'How many players from one team are on the court in volleyball?', 6::DOUBLE PRECISION),
        (55, 'Сколько игроков одной команды на поле в футболе?', 'How many players from one team are on the field in football?', 11::DOUBLE PRECISION),
        (56, 'Сколько игроков одной команды на льду в хоккее с шайбой?', 'How many players from one team are on the ice in ice hockey?', 6::DOUBLE PRECISION),
        (57, 'Сколько игроков одной команды на поле в бейсболе?', 'How many players from one team are on the field in baseball?', 9::DOUBLE PRECISION),
        (58, 'Высота баскетбольного кольца в футах?', 'What is the height of a basketball hoop in feet?', 10::DOUBLE PRECISION),
        (59, 'Длина марафонской дистанции в метрах?', 'What is the marathon distance in meters?', 42195::DOUBLE PRECISION),
        (60, 'Сколько видов спорта в современном пятиборье?', 'How many sports are in the modern pentathlon?', 5::DOUBLE PRECISION),
        (61, 'Сколько дисциплин в десятиборье?', 'How many events are in the decathlon?', 10::DOUBLE PRECISION),
        (62, 'Сколько дисциплин в семиборье?', 'How many events are in the heptathlon?', 7::DOUBLE PRECISION),
        (63, 'Сколько таймов в футбольном матче?', 'How many halves are in a football match?', 2::DOUBLE PRECISION),
        (64, 'Сколько минут длится один тайм в футболе?', 'How many minutes are in one half of a football match?', 45::DOUBLE PRECISION),
        (65, 'Сколько турниров Большого шлема по теннису проводится в год?', 'How many Grand Slam tennis tournaments are held each year?', 4::DOUBLE PRECISION),
        (66, 'Сколько колёс у болида Формулы-1?', 'How many wheels does a Formula 1 car have?', 4::DOUBLE PRECISION),
        (67, 'До скольких очков обычно играют сет в волейболе?', 'To how many points is a standard volleyball set usually played?', 25::DOUBLE PRECISION),
        (68, 'До скольких очков играют тай-брейк сет в волейболе?', 'To how many points is a volleyball tie-break set played?', 15::DOUBLE PRECISION),
        (69, 'Сколько иннингов в стандартном бейсбольном матче?', 'How many innings are in a regulation baseball game?', 9::DOUBLE PRECISION),
        (70, 'Сколько стоек в одной калитке в крикете?', 'How many stumps are in one cricket wicket?', 3::DOUBLE PRECISION),
        (71, 'Сколько лунок в стандартном раунде гольфа?', 'How many holes are in a standard round of golf?', 18::DOUBLE PRECISION),
        (72, 'В каком году открыли Плутон?', 'In what year was Pluto discovered?', 1930::DOUBLE PRECISION),
        (73, 'Сколько естественных спутников у Земли?', 'How many natural satellites does Earth have?', 1::DOUBLE PRECISION),
        (74, 'Сколько миллионов километров примерно в одной астрономической единице?', 'How many million kilometers are approximately in one astronomical unit?', 150::DOUBLE PRECISION),
        (75, 'Сколько минут примерно идёт свет от Солнца до Земли?', 'About how many minutes does sunlight take to reach Earth?', 8::DOUBLE PRECISION),
        (76, 'Сколько континентов обычно выделяют в школьной географии?', 'How many continents are usually taught in school geography?', 7::DOUBLE PRECISION),
        (77, 'Сколько океанов обычно выделяют на Земле?', 'How many oceans are usually recognized on Earth?', 5::DOUBLE PRECISION),
        (78, 'Глубина Бездны Челленджера в метрах, округлённо?', 'What is the depth of Challenger Deep in meters, rounded?', 10994::DOUBLE PRECISION),
        (79, 'При какой температуре по Цельсию вода замерзает?', 'At what temperature in Celsius does water freeze?', 0::DOUBLE PRECISION),
        (80, 'Абсолютный ноль в градусах Цельсия, округлённо?', 'What is absolute zero in degrees Celsius, rounded?', -273::DOUBLE PRECISION),
        (81, 'Скорость звука в воздухе при 20 °C в м/с, округлённо?', 'What is the speed of sound in air at 20 °C in m/s, rounded?', 343::DOUBLE PRECISION),
        (82, 'Атомный номер углерода?', 'What is the atomic number of carbon?', 6::DOUBLE PRECISION),
        (83, 'Атомный номер кислорода?', 'What is the atomic number of oxygen?', 8::DOUBLE PRECISION),
        (84, 'Атомный номер золота?', 'What is the atomic number of gold?', 79::DOUBLE PRECISION),
        (85, 'Атомный номер железа?', 'What is the atomic number of iron?', 26::DOUBLE PRECISION),
        (86, 'Сколько атомов в молекуле воды?', 'How many atoms are in a water molecule?', 3::DOUBLE PRECISION),
        (87, 'Сколько электронов у нейтрального атома водорода?', 'How many electrons are in a neutral hydrogen atom?', 1::DOUBLE PRECISION),
        (88, 'Сколько протонов в ядре гелия?', 'How many protons are in a helium nucleus?', 2::DOUBLE PRECISION),
        (89, 'Какой pH у нейтральной воды?', 'What is the pH of neutral water?', 7::DOUBLE PRECISION),
        (90, 'Сколько хромосом у человека?', 'How many chromosomes do humans have?', 46::DOUBLE PRECISION),
        (91, 'Сколько пар хромосом у человека?', 'How many pairs of chromosomes do humans have?', 23::DOUBLE PRECISION),
        (92, 'Сколько зубов у взрослого человека обычно?', 'How many teeth does an adult human usually have?', 32::DOUBLE PRECISION),
        (93, 'Сколько молочных зубов у ребёнка?', 'How many baby teeth does a child have?', 20::DOUBLE PRECISION),
        (94, 'Сколько камер в сердце человека?', 'How many chambers are in the human heart?', 4::DOUBLE PRECISION),
        (95, 'Сколько групп крови в системе ABO?', 'How many blood groups are in the ABO system?', 4::DOUBLE PRECISION),
        (96, 'Сколько рёбер у взрослого человека?', 'How many ribs does an adult human have?', 24::DOUBLE PRECISION),
        (97, 'Сколько шейных позвонков у человека?', 'How many cervical vertebrae does a human have?', 7::DOUBLE PRECISION),
        (98, 'Нормальная температура тела человека в градусах Цельсия, округлённо?', 'What is normal human body temperature in degrees Celsius, rounded?', 37::DOUBLE PRECISION),
        (99, 'Сколько недель обычно длится беременность человека?', 'How many weeks does a typical human pregnancy last?', 40::DOUBLE PRECISION),
        (100, 'Сколько периодов в периодической таблице?', 'How many periods are in the periodic table?', 7::DOUBLE PRECISION),
        (101, 'Сколько групп в периодической таблице?', 'How many groups are in the periodic table?', 18::DOUBLE PRECISION),
        (102, 'Сколько стандартных азотистых оснований в ДНК?', 'How many standard nitrogenous bases are in DNA?', 4::DOUBLE PRECISION),
        (103, 'Сколько стандартных азотистых оснований в РНК?', 'How many standard nitrogenous bases are in RNA?', 4::DOUBLE PRECISION),
        (104, 'Сколько всего кодонов в генетическом коде?', 'How many codons are in the genetic code?', 64::DOUBLE PRECISION),
        (105, 'Сколько стоп-кодонов в генетическом коде?', 'How many stop codons are in the genetic code?', 3::DOUBLE PRECISION),
        (106, 'Сколько стандартных аминокислот кодирует генетический код?', 'How many standard amino acids does the genetic code encode?', 20::DOUBLE PRECISION),
        (107, 'Сколько базовых единиц в системе СИ?', 'How many SI base units are there?', 7::DOUBLE PRECISION),
        (108, 'В каком году был запущен первый искусственный спутник Земли?', 'In what year was the first artificial Earth satellite launched?', 1957::DOUBLE PRECISION),
        (109, 'Сколько пилотируемых высадок на Луну было в программе Apollo?', 'How many crewed Moon landings were there in the Apollo program?', 6::DOUBLE PRECISION),
        (110, 'Примерно за сколько минут МКС совершает один оборот вокруг Земли?', 'About how many minutes does the ISS take to orbit Earth once?', 90::DOUBLE PRECISION),
        (111, 'В каком году была принята Декларация независимости США?', 'In what year was the United States Declaration of Independence adopted?', 1776::DOUBLE PRECISION),
        (112, 'В каком году началась Французская революция?', 'In what year did the French Revolution begin?', 1789::DOUBLE PRECISION),
        (113, 'В каком году пала Берлинская стена?', 'In what year did the Berlin Wall fall?', 1989::DOUBLE PRECISION),
        (114, 'В каком году была подписана Великая хартия вольностей?', 'In what year was Magna Carta sealed?', 1215::DOUBLE PRECISION),
        (115, 'В каком году была принята Всеобщая декларация прав человека?', 'In what year was the Universal Declaration of Human Rights adopted?', 1948::DOUBLE PRECISION),
        (116, 'В каком году была основана ООН?', 'In what year was the United Nations founded?', 1945::DOUBLE PRECISION),
        (117, 'В каком году было основано НАТО?', 'In what year was NATO founded?', 1949::DOUBLE PRECISION),
        (118, 'В каком году вступил в силу Маастрихтский договор?', 'In what year did the Maastricht Treaty enter into force?', 1993::DOUBLE PRECISION),
        (119, 'В каком году евро ввели в наличное обращение?', 'In what year was euro cash introduced?', 2002::DOUBLE PRECISION),
        (120, 'В каком году произошла авария на Чернобыльской АЭС?', 'In what year did the Chernobyl disaster happen?', 1986::DOUBLE PRECISION),
        (121, 'В каком году братья Райт совершили первый управляемый полёт?', 'In what year did the Wright brothers make the first controlled flight?', 1903::DOUBLE PRECISION),
        (122, 'В каком году примерно появился печатный станок Гутенберга?', 'Around what year did Gutenberg''s printing press appear?', 1440::DOUBLE PRECISION),
        (123, 'В каком году был отправлен первый email?', 'In what year was the first email sent?', 1971::DOUBLE PRECISION),
        (124, 'В каком году Тим Бернерс-Ли предложил World Wide Web?', 'In what year did Tim Berners-Lee propose the World Wide Web?', 1989::DOUBLE PRECISION),
        (125, 'В каком году появился первый веб-сайт?', 'In what year did the first website appear?', 1991::DOUBLE PRECISION),
        (126, 'В каком году впервые выпустили ядро Linux?', 'In what year was the Linux kernel first released?', 1991::DOUBLE PRECISION),
        (127, 'В каком году был выпущен язык Java?', 'In what year was Java released?', 1995::DOUBLE PRECISION),
        (128, 'В каком году был создан язык C?', 'In what year was the C programming language created?', 1972::DOUBLE PRECISION),
        (129, 'В каком году был создан JavaScript?', 'In what year was JavaScript created?', 1995::DOUBLE PRECISION),
        (130, 'В каком году появилась первая версия Unix?', 'In what year did the first version of Unix appear?', 1969::DOUBLE PRECISION),
        (131, 'В каком году был изобретён Ethernet?', 'In what year was Ethernet invented?', 1973::DOUBLE PRECISION),
        (132, 'В каком году вышел стандарт USB 1.0?', 'In what year was the USB 1.0 standard released?', 1996::DOUBLE PRECISION),
        (133, 'В каком году изобрели QR-код?', 'In what year was the QR code invented?', 1994::DOUBLE PRECISION),
        (134, 'В каком году была запущена Wikipedia?', 'In what year was Wikipedia launched?', 2001::DOUBLE PRECISION),
        (135, 'В каком году был запущен YouTube?', 'In what year was YouTube launched?', 2005::DOUBLE PRECISION),
        (136, 'В каком году был запущен Twitter?', 'In what year was Twitter launched?', 2006::DOUBLE PRECISION),
        (137, 'В каком году был запущен Instagram?', 'In what year was Instagram launched?', 2010::DOUBLE PRECISION),
        (138, 'Сколько серий в первом сезоне сериала Игра престолов?', 'How many episodes are in the first season of Game of Thrones?', 10::DOUBLE PRECISION),
        (139, 'Сколько книг в серии о Гарри Поттере?', 'How many books are in the Harry Potter series?', 7::DOUBLE PRECISION),
        (140, 'Сколько фильмов в оригинальной трилогии Звёздных войн?', 'How many films are in the original Star Wars trilogy?', 3::DOUBLE PRECISION),
        (141, 'Сколько книг входит в трилогию Властелин колец?', 'How many books are in The Lord of the Rings trilogy?', 3::DOUBLE PRECISION),
        (142, 'Сколько клеток на поле игры Монополия?', 'How many spaces are on a Monopoly board?', 40::DOUBLE PRECISION),
        (143, 'Сколько ячеек в судоку 9 на 9?', 'How many cells are in a 9 by 9 sudoku?', 81::DOUBLE PRECISION),
        (144, 'Сколько клеток в одном малом квадрате судоку 3 на 3?', 'How many cells are in one 3 by 3 sudoku box?', 9::DOUBLE PRECISION),
        (145, 'Какой множитель даёт поле triple word в Scrabble?', 'What multiplier does a triple word square give in Scrabble?', 3::DOUBLE PRECISION),
        (146, 'Сколько байтов в килобайте в двоичном смысле?', 'How many bytes are in a binary kilobyte?', 1024::DOUBLE PRECISION),
        (147, 'Сколько килобайт в мегабайте в двоичном смысле?', 'How many kilobytes are in a binary megabyte?', 1024::DOUBLE PRECISION),
        (148, 'Сколько секунд в одном часе?', 'How many seconds are in one hour?', 3600::DOUBLE PRECISION),
        (149, 'Сколько секунд в сутках?', 'How many seconds are in a day?', 86400::DOUBLE PRECISION),
        (150, 'Сколько минут в сутках?', 'How many minutes are in a day?', 1440::DOUBLE PRECISION)
),
updated_existing AS (
    UPDATE game_question AS q
    SET question_text = s.question_text_ru,
        question_text_ru = s.question_text_ru,
        question_text_en = s.question_text_en,
        correct_answer = s.correct_answer,
        is_active = TRUE,
        updated_at = NOW()
    FROM ordinary_seed AS s
    WHERE q.game_type = 'number_duel'
      AND q.question_text_ru = s.question_text_ru
    RETURNING s.position
)
INSERT INTO game_question (uid, game_type, question_text, question_text_ru, question_text_en, correct_answer, is_active)
SELECT ('33333333-3333-4333-8333-' || LPAD(s.position::TEXT, 12, '0'))::UUID,
       'number_duel',
       s.question_text_ru,
       s.question_text_ru,
       s.question_text_en,
       s.correct_answer,
       TRUE
FROM ordinary_seed AS s
WHERE NOT EXISTS (
    SELECT 1
    FROM updated_existing AS u
    WHERE u.position = s.position
);
