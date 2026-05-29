DO $$
DECLARE
  pwd  CONSTANT TEXT := '$2a$10$OcupBh7XKZN3d7qZWmWMP.rNYvhJobBS4xhdYMd4sPztorUUI3VlO';
  i         INT;
  j         INT;
  liker_idx INT;
  acc_id    BIGINT;
  prof_id   BIGINT;
  media_id  BIGINT;
  cover_id  BIGINT;
  post_id   BIGINT;
  comm_id   BIGINT;
  cprof_id  BIGINT;

  prof_ids     BIGINT[] := '{}';
  cprof_ids    BIGINT[] := '{}';
  comm_ids     BIGINT[] := '{}';
  post_ids     BIGINT[] := '{}';
  post_authors BIGINT[] := '{}';

  fnames TEXT[] := ARRAY[
    'Александр','Михаил','Дмитрий','Иван','Алексей',
    'Андрей','Сергей','Николай','Владимир','Артём',
    'Кирилл','Павел','Максим','Роман','Евгений',
    'Антон','Денис','Илья','Тимур','Даниил',
    'Константин','Вадим','Виктор','Олег','Игорь',
    'Анастасия','Мария','Екатерина','Светлана','Ольга',
    'Наталья','Елена','Юлия','Дарья','Полина',
    'Виктория','Алина','Татьяна','Ирина','Ксения',
    'Валерия','Вера','Надежда','Людмила','Тамара',
    'Нина','Галина','Лариса','Зинаида','Регина'
  ];
  lnames TEXT[] := ARRAY[
    'Иванов','Петров','Сидоров','Козлов','Новиков',
    'Морозов','Соколов','Волков','Попов','Лебедев',
    'Смирнов','Орлов','Федоров','Зайцев','Макаров',
    'Кузнецов','Тихонов','Гусев','Зимин','Борисов',
    'Семёнов','Степанов','Ершов','Беляев','Власов',
    'Иванова','Петрова','Сидорова','Козлова','Новикова',
    'Морозова','Соколова','Волкова','Попова','Лебедева',
    'Смирнова','Орлова','Федорова','Зайцева','Макарова',
    'Кузнецова','Тихонова','Гусева','Зимина','Борисова',
    'Семёнова','Степанова','Ершова','Беляева','Власова'
  ];
  genders TEXT[] := ARRAY[
    'male','male','male','male','male',
    'male','male','male','male','male',
    'male','male','male','male','male',
    'male','male','male','male','male',
    'male','male','male','male','male',
    'female','female','female','female','female',
    'female','female','female','female','female',
    'female','female','female','female','female',
    'female','female','female','female','female',
    'female','female','female','female','female'
  ];
  towns TEXT[] := ARRAY[
    'Москва','Санкт-Петербург','Новосибирск','Екатеринбург','Казань',
    'Нижний Новгород','Челябинск','Самара','Уфа','Ростов-на-Дону',
    'Красноярск','Пермь','Воронеж','Волгоград','Краснодар',
    'Саратов','Тюмень','Тольятти','Ижевск','Барнаул',
    'Ульяновск','Хабаровск','Махачкала','Иркутск','Томск',
    'Москва','Санкт-Петербург','Новосибирск','Екатеринбург','Казань',
    'Нижний Новгород','Челябинск','Самара','Уфа','Ростов-на-Дону',
    'Красноярск','Пермь','Воронеж','Волгоград','Краснодар',
    'Саратов','Тюмень','Тольятти','Ижевск','Барнаул',
    'Ульяновск','Хабаровск','Махачкала','Иркутск','Томск'
  ];
  bios TEXT[] := ARRAY[
    'Люблю технологии и разработку',
    'Студент физического факультета',
    'Программист на Python и Go',
    'Путешествую при каждой возможности',
    'Спорт и здоровый образ жизни',
    'Фотограф-любитель',
    'Читаю книги по истории',
    'Играю на гитаре уже пять лет',
    'Работаю в сфере финансов',
    'Дизайнер интерфейсов',
    'Увлекаюсь астрономией',
    'Кулинарный энтузиаст',
    'Пишу стихи в свободное время',
    'Занимаюсь боксом',
    'Преподаю математику',
    'Стартапер и предприниматель',
    'Фанат научной фантастики',
    'Велосипедист и турист',
    'Учусь на факультете журналистики',
    'Разрабатываю мобильные приложения',
    'Художник и иллюстратор',
    'Работаю в медицине',
    'Люблю настольные игры',
    'Занимаюсь волонтёрством',
    'Интересуюсь экологией и природой'
  ];

  cunames TEXT[] := ARRAY[
    'seedtech','seedsci','seedsport','seedart','seedgame',
    'seedmusic','seedfilm','seedtravel','seedfood','seedphoto'
  ];
  ctitles TEXT[] := ARRAY[
    'Технологии','Наука','Спорт','Искусство','Игры',
    'Музыка','Кино','Путешествия','Кулинария','Фотография'
  ];
  cbios TEXT[] := ARRAY[
    'Обсуждаем новейшие технологии и разработки',
    'Наука вокруг нас: открытия и исследования',
    'Спортивные новости и достижения',
    'Искусство во всех его проявлениях',
    'Игровое сообщество для всех платформ',
    'Музыка без границ и жанров',
    'Кино и сериалы для всех',
    'Впечатления и маршруты путешественников',
    'Рецепты и кулинарные эксперименты',
    'Фотографии и советы по съёмке'
  ];

  user_avatars TEXT[] := ARRAY[
    'https://www.rupixel.ru/files/preview/1280x852/13951760165587yonhoep4rqdrxsmhcrzprrwjlknopkgzve2mloni16ts4go31gmm7wnwbbrgkgxmydwsv4yysd10gom1wamezclzr9iwrjsktirs.jpg',
    'https://www.rupixel.ru/files/preview/960x1436/14201761353608pdmdmhtgz8ha7vydjr1n9aq7060ievaqhvhl8govtewrzpq6w8mlxcuovi4uol92xoftbmu3grj0bi90mdn5iwvzxgmpvhnh1emz.jpeg',
    'https://www.rupixel.ru/files/preview/1280x853/21723099259wchixlvmshw1swj2znfvj0lldor2lu1uuy9rgempdtgiaiqoskgopmgyasoxp3bxufuhmhzxsrwhltlijynhwuswglabjin4f2si.jpg',
    'https://www.rupixel.ru/files/preview/1280x1049/21723098980onza7yaq04mo0svrpuk0pwtxwy2ancqzu7nbtqjzbboez7ir7jkjwpx1rt8zujioqys0tszkqxqukcyygvonk1cfaea6uvyion4z.jpg',
    'https://www.rupixel.ru/files/preview/960x1280/17511777379788lrqzdgs3hbwgzsqb8a1lw8bkvooimelr4jbfmaxn94xh38bdh3y2qmigusfbnxjalq08i1qqhlhxvowjl9cef1oggnxcp872zxpt.jpg',
    'https://www.rupixel.ru/files/preview/960x1280/17511777201898urfc4djbxgnx4686cnj7yjsiktrz80qqurfzuu8yeues37lepabaawoszhozihaacwkdwes9kxp3e6wxx8yur3hiq9j6vod8fuy9.jpg',
    'https://www.rupixel.ru/files/preview/960x1440/21725605582yowh8dffoahibmyapnogr4e0ux2qgltoinsbedvsbmv0bjgignh55jmemuvkkeswv1vnuun0f7tfbtgbzlmn5eme8tvg73cafkux.jpg',
    'https://www.rupixel.ru/files/preview/961x1442/21724309808uevpdo5lvktdz56vnpqmvnvsbxt4t6drbt3q4paolnslziawj5q1veppvrytv8g02nnqvxgnqz0ywr6wbrzfujajlzjejkes2q1m.jpg',
    'https://www.rupixel.ru/files/preview/960x1200/11761744438610bizxnm2vwyhvh2m2j8v1fiim479i3cnxa0r8poabotxw0svzfof5wwqkoese137q1eascnxn8fiuo7xcmufuqodr4aohzgstw3jh.jpg',
    'https://www.rupixel.ru/files/preview/960x1200/11761744438483fm8qngjrx6exqk4jolrgmnwxovzvpmm8a8se50qqcpzasnacxwcmxhxahlt9nk5jjwotalgry2wrc5vltpnqbsaz9mmrdu0sfhlg.jpg'
  ];
  post_imgs TEXT[] := ARRAY[
    'https://www.rupixel.ru/files/preview/1280x854/25917381426237m4dhsbe3vrb6bfe3drr2tkekgeowqmoeguiigrjgq7wdpewtbvt019toet0efqosgc4muqa4pslnhuudcyvciyacjyx2w1md5je.jpg',
    'https://www.rupixel.ru/files/preview/1280x854/1131676876825xq6moieglh4uqkjzhlcb4sh9zzqlowcwshbfjjkxzosfx5ejdgwnyvzysqphw2dhbftegkkguec5fonpx0grtljtmxbdynqceyhp.jpg',
    'https://www.rupixel.ru/files/preview/1281x854/21665134214lt83cbzqwwhimhbmpdecfxbmd2olfdx396teysopb7mjktuaqlqlbbpu0198ofzflbnflmsfflri7s30kcunpcpsnellgdedr9ck.jpg',
    'https://www.rupixel.ru/files/preview/960x1280/21667204381tbdsyspbjalk31tashq6h9nxuc7okap2jmfy0cpmr5ylggqopzh3hm0tuk3qyrssodimhesnvdzcwnownagwhupy1c2ee4occwhy.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21665041509ppznk3pdc0oh5cgbmxz9vsawcmwlno9e1he7ej9twbzwljtdag2phadbeuqay3jmf3bufnjozexjxdunbc4xaznwojf9th2lcqsf.jpg',
    'https://www.rupixel.ru/files/preview/1280x849/21665040050uqdqp7v5qhhzuclikawj5m1v2vxztl8cipumvhr2nfyqdk4zduryie9ylai0af401xadpugdao3jphrd6y0zgrxugamj0einkslm.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21670412529wqjrtoivojkzypuykk5dpxqk4lzrekranbf7vfsukoepimtrlyw72qeh7wkyvkzeymxcregi3japikynmxjfymwygsywjaked85g.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21723096975spa3zy4faagfxmc1hm8kghlcxe6acw6sqgsmgyku5kv4fg1rgheswy37tyo0qenxgvusrrzhx2hi9cjakcm28dctwx8np2kzvnie.jpg',
    'https://www.rupixel.ru/files/preview/1280x960/216942693755rc03iq6avo1wamj3005tzdxbdbos5qtrj4uijf5fqspwfoovoxspihlblxtdzd9xykarowcf2lylnioczk4zoxx3guosfejttwz.jpg',
    'https://www.rupixel.ru/files/preview/1280x991/216651340498uwgg30w3s7ywcwvomym83krdfh29ncxsboensbs6hjcdgagy9vxdvdyd6deasnqszzg52wfpvjp4edu1uzu2uoo7c7qqtwabpbj.jpg',
    'https://www.rupixel.ru/files/preview/1280x854/2591698235989nizn8afw3pvgbd2kclwc0emt1hdmijmlft7n50p7lyp3hhqtwtxreg7g1yjwuxnhjruowjqtq2lygigs542z6ndfrjpzlnj7dwdg.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/216651304371c9z62qe9d2n0zzaye6kzdugjqbcmj5upyup02exjixk6ig9cjcri165jmehwu0qyznhazwdxbydoqdatavjvnnkyccqb46fcckd.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21670521887eggrlmpjjpvuaua3itn1wnrwgijdfxfttg7nr06stcjnprc2vaq0ds6jie7wcnikpct3xfk6bjbsfgrljnqjecvzjjysnenfl9wg.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/6271731770719unf11zjqbuqetskut9e5d71ji50qdoanuwnd40lssfmzkwipjz5hhc6lka8zg38w3i4f0zvz0ytuhdavtdtp5e8sgquprx2yu066.jpg',
    'https://www.rupixel.ru/files/preview/1280x854/21675924177x0x75rngcilwjbnhmcrb2jmmiwacmdxmy8wwaydxdaukslplfikensyxyijkdqok1az8ultgrgigyqfokbh9cnhedo3dfoqev4lg.jpg',
    'https://www.rupixel.ru/files/preview/1280x848/21694754752fcqpud0d38j3l1vt0sknp7gneyqfqojdqo0pv3ui6n9doq58rxeyjbp5gxfgrs3ta5ou1egodmilwvapg3zdpgbuw05dfnpj6xcv.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21665041384leufkvcxjw3i4lmfwlcam1kv0jte5fzo3amwldlcjvnxku92gdhfjqe0l83dvyf2wz88mtuksxwjzetmpoiyofauuhlfx1j2tdji.jpg',
    'https://www.rupixel.ru/files/preview/1280x720/17551777355335f5nffqmi6idiyirjqggs8nmw4cvpw9nsdjtl3al1xbdh1jbogjwrq19noaggrukvqwz6wbz6oai4xwgtsdotuflvi8ykx7nwpxas.jpeg',
    'https://www.rupixel.ru/files/preview/1280x848/21665041381vjmp3wz12r3bkahuiyivxlelcasrdedpaxg18u1kadzmc4lctsddydhzaqi2xbwqp3bdrg1do8mxudmyffys8qci6jcg1z37kzl8.jpg',
    'https://www.rupixel.ru/files/preview/1280x875/216947488251ymhccnbywqnh7k4hdbhpusgifo5snlcqq5bkvpywnmxdp8mq4se1jqucnmfwgqqtrgu4qvhifzder9sckfsadsb5qwkptvebjji.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21665041693ii5iwfborra07shc0h3xmavgyhf7hpztnxe5grw7bvpnhyaqgy3kbgofyjwmgr0gwvmwzzm6thalvutusw5rwq1z2g4fdcqmjngn.jpg',
    'https://www.rupixel.ru/files/preview/1280x960/216704292899pdzpvdpfpt5k57psnj5e3yqna2hjqgm1p3tdfvsww1z0corqfdorwtefdu2wzksnpfjisp5pjyckldynx3jiwm897i0d220cyrq.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21665133949iwztjx5zjy2htmqjemgumhvwysguzgtxbqzcfabveuvy8zxnzgp2vgxtkn3lj6dtsqq6b04bdldg3kh5ca8hgfn9uy3ovjezklk5.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/15431770966585kqdevtue703zigkisxsdm8bplhbcmejmj99tualwayftxahqxxdksk6hqsgwmnwfyejl553nh1i1h0muey4svndsfjijgxhxfi9l.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/216650398278ucbm0qcvnyok88yhh0bhqkvwozrparet7lnjnhstj53b9wmkwfoh23lokotshruga7pydrjm1gb52yxv9z7lymotoxghxnkdnzo.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21670479438kpiml4mirpf4fj8te5uezb8hmn0h6z2zwkwlttijxo1ou9hezfw3gzop5euk3ooumqncbia50ifqnma26xz2hurpfr90ln2ulyav.jpg',
    'https://www.rupixel.ru/files/preview/1280x849/21678102010ulktskgbrunj0g2ywnnrfv0l31nfoo2udyfw08by0orgu25yvyw65ftgbndzmfmuroj3ierk1kfrrityvphh5b64jee3zo7h4k1b.jpg',
    'https://www.rupixel.ru/files/preview/1280x852/21694748062jfjizjmkzqnxwbcqmtunyqqvrllia1qfzig3g6zbc6xuhomjpjnld4tphc5isssqqgstj8e1on0gip6xqv3b7suetkbspu2ua144.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21694269626w8hoifjmquptouyl9q8b30cp5tvtaxvkdiwftlcgrh0grupb3pb2ja1el7l5zqaszsaqrxtwz1k4dbnxvm761e2ju3gocgxqxh39.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/216651335916vjcwzxuvxhr6ztlqoguku7cnv8tescuc0kwpm5ybqywa0keztdhnicanlvutl63oka8bliitg5ewdvaqvllrceenikpurvvesit.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21694754751ok4zyml8khwtvv8rja7kjtofdh2tfjn4udnzijkfbnlc6i53wt6hkmq6udrihrjcbvlmsn1eccbgnutaoeff73ijoykeuqzdgwft.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/216759236300sszbqpvsocmwxtzmjgerktijeldlcgydhhfmrbameh4k968vlevit6eae3w6jokclbkq3vffvtjkln4dalakp6aho3gcixpekfa.jpg',
    'https://www.rupixel.ru/files/preview/1280x960/21670434443yarwlj7ihbpe9svnhurezlahikcv2pbagzsleploxlvapdjqpvtjpmu8v2u9ybbdtwjqkllpwa5zipx7e4nwnk2gkogpzg7hdapa.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/14471779121600yoqku4ww0otr1gxrgxcnlsoi9lawuapuhgi1hsgz3p9xybki0raprac8qtkxbnvjgx2daiiy3qoetejmekza3lfgdogjlbkaswnc.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21665133535zjhjyzu4xawnqzqb53cvqiyd3i6ey91ifaiixboscw0uwk24uchuwrxbflj8c2jlts4mcr4pb7hx0eplk4jgoub1bndq3wijmm5p.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21694754639kkytz5ucwxbjjevnhik9wgfgknioqaykizvkuenmulsrdo0i0auncdj8siwewwvurdrgjsiheigwo6pdsga0o28uejo6rrkpkvvo.jpg',
    'https://www.rupixel.ru/files/preview/960x1440/216705720454n6rjgqoyj0acaigkcmikzsttjdrm9hemgfip9pedxdoftg4vijo4agzqjckasswdi6mqtelgldh8mfk7qwg7xm6gdu3rksfok5i.jpeg',
    'https://www.rupixel.ru/files/preview/1280x720/175517773530247ehds5bhu98wo7jh9dh3yglqzulrdzugs1rven2sm0plibez6hwavholvhbnhokhfpo8k7n6r8nib0thigpemu5pdwsz2mjzfroz.jpeg',
    'https://www.rupixel.ru/files/preview/1280x911/21678435594z7peco0eqneoqq44tgqc55yk8zlv55rm01sawkyfatkonchvxo63wfpu5eh0jjhbo4t3im9hz97zaq7pbst8c0nuzetarh7a7tbd.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/216947507271sdwvjmv0lrctdjevtwygrwu0mxjux8wdwsopbkbbsxokxpb1v3thgjvhfpuauu59kxagny8qdz9tqsal1qhatj6ogaplhncs9m8.jpg',
    'https://www.rupixel.ru/files/preview/1280x960/216704830628r2ih6eickhaxpizoa6kudayxnni4lbtr0ohsyd1ich9t6dige4npxt12b9btvw11bnagc9ykdxusphnifzzddtfbqni6hbm5r4u.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/1131676878999vpxzl7pw7rj5jmd1uehmfkxfufktopupbjcscdjbts5w7gs2ncao2bkw8inbjbvhial3gd8e73hvu4ilhl2n0hq22jzggha7xbbm.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/216651335353xyilurnnu3qde9l1l9l6vypfiqptnfnwx8dvual5juujls2oaeilbvwqsd3ac9m8y7gdtd7q8mlodi0rinioygtxbvccjnqffh9.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/154317709667035lrvi8uif8eo1n8vjrmckhotxsbxmvehlnnmjjuclhwennkozhwm7qnjugjtm47crojfnpfj8dczsos0depnqn7epgpilyagskuc.jpg',
    'https://www.rupixel.ru/files/preview/1280x850/21664946893ccajwqdjbfzisfsdocmxbifdirdgcxdndxyojo8eetimd9cvnymes72zbrsptdp1zey3fgxpa0tendpfnalys19sjen1aqrixe7f.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/10941734810958mmnkxrt7pa2oogp0omipmcs6gf5npxw1xyb34o22eniyfpkv0rpa9ohtpmsxy3javooeoaiwcrjlkwxcdoynpikssfqayeczrfxp.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21670483067wvcx7xvrzudy7hbovfwfbfdanpufyibgou08eiisxpczd8sgpjogob3swcv92zy7gqygkqahvn0wccz8ckryfv1grxdmqmozvcoy.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21694270007bm3cjqugli8hcmp6z0tayglmtxzyumsuzb0bkgfkncm2kanowixycf46ftsgbh0mgemcb1vr91okqqwowwx4kpe8offc1ww91i7i.jpg',
    'https://www.rupixel.ru/files/preview/960x1440/144717630202802fz3fxaw8hihgxybynstgtfhxpxasuybstzq938piu7phc4gej1rx3hn4hueu6pznvgelguprnj1nsjuveqbb3vgtvldj4xyvygp.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21665133595vosgob5bg9lnvdklwbwitwiwpjn3uil3mnjbakqgzrl8f1z4sqscjtxnas8gy99ftkccdgdx6mogx86kknqobsuhl0gugsuoncw0.jpg'
  ];
  comm_covers TEXT[] := ARRAY[
    'https://www.rupixel.ru/files/preview/1280x876/21670431258pisyrxmi2sxdpfkn8vxztxsbrctvdzm1xiqi7x6e80xcx0tc9xl0neocmgt9p663lfhfkp0too1hmhpbm9rspdbyfbydu12eryhm.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21670412529wqjrtoivojkzypuykk5dpxqk4lzrekranbf7vfsukoepimtrlyw72qeh7wkyvkzeymxcregi3japikynmxjfymwygsywjaked85g.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/217230968551lplpuobd3oixqyxjqoa4kvwj2ygtmtw1uc7e7q8tbj1fxj07njdgwcl8f9qvs3iv71jzhxzlcfkfawafeuh91ngzzt8ylgmzcrf.jpg',
    'https://www.rupixel.ru/files/preview/1280x854/21675924177x0x75rngcilwjbnhmcrb2jmmiwacmdxmy8wwaydxdaukslplfikensyxyijkdqok1az8ultgrgigyqfokbh9cnhedo3dfoqev4lg.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/216705009427k1mnyjdncnncidfofpa0bdw7d2dv1i6kxsvo5lgbyfmxxtdm5p1av5dk89evdqvfyxsoznyqchgv6lgaf06fkxpb9uunpf4vgxz.jpg',
    'https://www.rupixel.ru/files/preview/1280x854/2591698235989nizn8afw3pvgbd2kclwc0emt1hdmijmlft7n50p7lyp3hhqtwtxreg7g1yjwuxnhjruowjqtq2lygigs542z6ndfrjpzlnj7dwdg.jpg',
    'https://www.rupixel.ru/files/preview/1280x849/21665040050uqdqp7v5qhhzuclikawj5m1v2vxztl8cipumvhr2nfyqdk4zduryie9ylai0af401xadpugdao3jphrd6y0zgrxugamj0einkslm.jpg',
    'https://www.rupixel.ru/files/preview/1280x960/101117217247735oxqiucevb2elob7ljosh9weahxtcsjwk30obgcypm1wpnk2upsn3b34nqhykidjjkjbas0bdpuimk728cdlqpekoxno5jz34atj.jpg',
    'https://www.rupixel.ru/files/preview/1280x720/175517773530247ehds5bhu98wo7jh9dh3yglqzulrdzugs1rven2sm0plibez6hwavholvhbnhokhfpo8k7n6r8nib0thigpemu5pdwsz2mjzfroz.jpeg',
    'https://www.rupixel.ru/files/preview/960x1131/11621777200641cogzpu3qbcc4ghomgzto22zp9xv4qqkgwsydnthsbwkt1gssawhjpj2n6z1psvf1tf6qcyvtgiyodnfvpuo2vj568pbw7gm5tvbw.jpg'
  ];

  comm_post_imgs TEXT[] := ARRAY[
    'https://www.rupixel.ru/files/preview/1280x848/21665041381vjmp3wz12r3bkahuiyivxlelcasrdedpaxg18u1kadzmc4lctsddydhzaqi2xbwqp3bdrg1do8mxudmyffys8qci6jcg1z37kzl8.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/216650398278ucbm0qcvnyok88yhh0bhqkvwozrparet7lnjnhstj53b9wmkwfoh23lokotshruga7pydrjm1gb52yxv9z7lymotoxghxnkdnzo.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21670435306alnekat7ew9uoackqgpbojkymmfm3zsrw8srujjxge2bhepua7hsvjzkij7vj8l4fzz8bt31gvqbkylfilgbkaibyx81zcawznik.jpg',
    'https://www.rupixel.ru/files/preview/1280x849/21678102010ulktskgbrunj0g2ywnnrfv0l31nfoo2udyfw08by0orgu25yvyw65ftgbndzmfmuroj3ierk1kfrrityvphh5b64jee3zo7h4k1b.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21670417021hdsbrgtxeexzmiuxfv5zty02g5m0atxgicadzmnrvwqjtelh1oysldiobcvtlbwiisyrqi9ymjv0mvsu55t5nfhyhyithmnml952.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21670483067wvcx7xvrzudy7hbovfwfbfdanpufyibgou08eiisxpczd8sgpjogob3swcv92zy7gqygkqahvn0wccz8ckryfv1grxdmqmozvcoy.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21694750906rd08o6iiwvan4cog7dejziquwthwcnfwvywla59f1xi5w66loqmnaetlta8odyr7fkkbigmtwrbtvwqgg1cxbszjjckeopmawnit.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21723096975spa3zy4faagfxmc1hm8kghlcxe6acw6sqgsmgyku5kv4fg1rgheswy37tyo0qenxgvusrrzhx2hi9cjakcm28dctwx8np2kzvnie.jpg',
    'https://www.rupixel.ru/files/preview/1280x823/216947511155wkpdyvicy1xjiqb2y4s6jmdvlp0nnowjjnkhzsr0bfno62owck5y5x7n9uqffkwf1v5ojschsf34s4w6zspfx9t7ir2vfnzy30n.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21665133535zjhjyzu4xawnqzqb53cvqiyd3i6ey91ifaiixboscw0uwk24uchuwrxbflj8c2jlts4mcr4pb7hx0eplk4jgoub1bndq3wijmm5p.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/216759236300sszbqpvsocmwxtzmjgerktijeldlcgydhhfmrbameh4k968vlevit6eae3w6jokclbkq3vffvtjkln4dalakp6aho3gcixpekfa.jpg',
    'https://www.rupixel.ru/files/preview/960x1280/21673601937v2igo2ps3ksz52son8eebqccnoceewmyl20bl3htk4vfnb0c41xle8cpzqfvdkoqrefym3fbwakbufbf2n9c9mmkyeheecrrn6n3.jpg',
    'https://www.rupixel.ru/files/preview/1280x720/216704314396abdbovbda7famwl37cu4wbccbxvhacdqrpjtsdyusrpzfcnzul1zv3jqlz3d2ayzyyr6j3nhzmjrrncqohhawrrrvsksssqbzkd.jpg',
    'https://www.rupixel.ru/files/preview/1280x719/21670435640bfzeztdyomgnb79dtts9gyqebr4xtpsfgb4koazh7e39efythnfuvkigr6kaompddc1oikwjodhhuimremkw5nkmqrsw0vhpona3.jpg',
    'https://www.rupixel.ru/files/preview/1280x848/216704359293wlekystty7f6blrtwba51azvzngfjfgqb92j2ir3trxm5v16yt3jpbtrxbpejw9s3qs4e4h28undck4kdbnow5ow6eo4nik70f5.jpg',
    'https://www.rupixel.ru/files/preview/1280x848/21694754752fcqpud0d38j3l1vt0sknp7gneyqfqojdqo0pv3ui6n9doq58rxeyjbp5gxfgrs3ta5ou1egodmilwvapg3zdpgbuw05dfnpj6xcv.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21694754751ok4zyml8khwtvv8rja7kjtofdh2tfjn4udnzijkfbnlc6i53wt6hkmq6udrihrjcbvlmsn1eccbgnutaoeff73ijoykeuqzdgwft.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21694754639kkytz5ucwxbjjevnhik9wgfgknioqaykizvkuenmulsrdo0i0auncdj8siwewwvurdrgjsiheigwo6pdsga0o28uejo6rrkpkvvo.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21670479438kpiml4mirpf4fj8te5uezb8hmn0h6z2zwkwlttijxo1ou9hezfw3gzop5euk3ooumqncbia50ifqnma26xz2hurpfr90ln2ulyav.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/10941734810958mmnkxrt7pa2oogp0omipmcs6gf5npxw1xyb34o22eniyfpkv0rpa9ohtpmsxy3javooeoaiwcrjlkwxcdoynpikssfqayeczrfxp.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/21670480261urrlosqr8euynnfeyfibax5x6fps3tkcbsu52w7zuqoxzbrqckvhhhdzw0zv9yqqrfsws82wwnkfdnbytdba22dtrapqe2e2oxxc.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/14471779121600yoqku4ww0otr1gxrgxcnlsoi9lawuapuhgi1hsgz3p9xybki0raprac8qtkxbnvjgx2daiiy3qoetejmekza3lfgdogjlbkaswnc.jpg',
    'https://www.rupixel.ru/files/preview/1280x854/1131676876825xq6moieglh4uqkjzhlcb4sh9zzqlowcwshbfjjkxzosfx5ejdgwnyvzysqphw2dhbftegkkguec5fonpx0grtljtmxbdynqceyhp.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/1131676878999vpxzl7pw7rj5jmd1uehmfkxfufktopupbjcscdjbts5w7gs2ncao2bkw8inbjbvhial3gd8e73hvu4ilhl2n0hq22jzggha7xbbm.jpg',
    'https://www.rupixel.ru/files/preview/1280x880/21726132926vytyhulqgvqdjinekc63pzfrnzk8tgnopehu62um89xvucknapbw6omc0338g7sa7dbzjqkcytnhvuyhgcrtfbdvitvygvhnbtp1.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/109417343670489sspwnvlazsxjwj9fx0ifljwg3i2jaiez3jmruapeifzkenefgkxslyq6smhhi8a01mlnwgyddlhsyrc0j5cljg20wefxmwewydf.jpg',
    'https://www.rupixel.ru/files/preview/1280x857/21665130275wkt8qucflcrhingwe8zfhcywzbsxqlwdeqxhgdaaiqjqjmwa3jszfwydoot3djo6sy1sqo2scjr86jxagvvwbpztbohyjxezrt2y.jpg',
    'https://www.rupixel.ru/files/preview/1280x850/21670483059pf3rnexsl4gomow4j1pxmc1uzhydarsfatsv4bwx8dmukglvmswootesrqge9wsssmrfzh9baspp0wd7beyogniy7kxmgfvtdgey.jpg',
    'https://www.rupixel.ru/files/preview/1280x853/216704357589jflnmgxdulienbxn299cnullvj4jxsiynpcjp7m1jtwhumxf4qt2ledmvy65ovy3wbo41wn7rgzxhjxvqkefzpsbkfujekog7vi.jpg',
    'https://www.rupixel.ru/files/preview/1280x960/21694271333wofoan4u1zgtythyqbmnzzlqc8ednshb8z364ffv2otnlpf4cqxkedhag6ok7cjybndfypj3jhlrg1ltxtudtrbig7tlqycevfex.jpg'
  ];

  ptexts TEXT[] := ARRAY[
    'Отличный день! Много работал над новым проектом.',
    'Только что вернулся из похода в горы. Виды потрясающие!',
    'Нашёл отличную книгу по алгоритмам. Рекомендую всем!',
    'Сегодня приготовил пасту карбонара. Получилось очень вкусно!',
    'Участвовал в хакатоне. Наша команда заняла второе место!',
    'Смотрел новый фильм. Очень понравился сюжет!',
    'Начал изучать машинное обучение. Сложно, но интересно!',
    'Сегодня пробежал десять километров. Личный рекорд!',
    'Встретился с друзьями. Давно так не смеялся!',
    'Закончил читать Войну и Мир. Великая книга!',
    'Новый альбом любимой группы просто огонь!',
    'Сделал ребрендинг своего портфолио. Как вам?',
    'Осваиваю новый язык программирования. Идёт неплохо!',
    'Сегодня закат был просто сказочный. Фото прилагается.',
    'Посетил выставку современного искусства. Вдохновляет!',
    'Первый урок игры на гитаре. Пальцы болят, но счастлив!',
    'Вернулся из командировки. Соскучился по дому.',
    'Новый рецепт смузи: банан, шпинат, имбирь. Советую!',
    'Ночь кодинга позади. Баг наконец пойман!',
    'Наш стартап получил первое финансирование!'
  ];
  ctexts TEXT[] := ARRAY[
    'Обсуждаем новые тренды — присоединяйтесь!',
    'Делимся открытиями этого года.',
    'Итоги сезона — ваши впечатления?',
    'Лучшие работы участников нашего сообщества.',
    'Топ этого месяца по версии сообщества.',
    'Плейлист недели от наших участников.',
    'Обзор лучших релизов этого года.',
    'Фотоотчёт с последней встречи.',
    'Новый конкурс — присоединяйтесь!',
    'Спасибо за активное участие!'
  ];
  cmts TEXT[] := ARRAY[
    'Отличный пост!',
    'Согласен полностью.',
    'Интересно, спасибо!',
    'Не знал об этом.',
    'Красиво!',
    'Подписываюсь под каждым словом.',
    'Тоже так думаю.',
    'Здорово!',
    'Спасибо за информацию.',
    'Очень актуально.',
    'Круто!',
    'Буду иметь в виду.',
    'Как всегда, отлично!',
    'Поддерживаю!',
    'Интересная точка зрения.',
    'Спасибо, познавательно.',
    'Любопытно!',
    'Продолжай в том же духе!',
    'Это вдохновляет.',
    'Замечательно!'
  ];

BEGIN
  IF EXISTS (SELECT 1 FROM user_account WHERE username = 'seed001') THEN
    RAISE NOTICE 'Seed data already present, skipping.';
    RETURN;
  END IF;

  -- ── Phase 1: 50 users ──────────────────────────────────────────────────────
  FOR i IN 1..50 LOOP
    INSERT INTO user_account (uid, email, password_hash, username)
    VALUES (
      gen_random_uuid(),
      'seed' || LPAD(i::TEXT, 3, '0') || '@aris.test',
      pwd,
      'seed' || LPAD(i::TEXT, 3, '0')
    )
    RETURNING id INTO acc_id;

    INSERT INTO profile (uid) VALUES (gen_random_uuid()) RETURNING id INTO prof_id;

    INSERT INTO media (uid, media_name, author_id, extension, mime_type, link, size)
    VALUES (
      gen_random_uuid(),
      'avatar_seed' || LPAD(i::TEXT, 3, '0'),
      prof_id, 'jpg', 'image/jpeg',
      user_avatars[((i - 1) % array_length(user_avatars, 1)) + 1],
      0
    )
    RETURNING id INTO media_id;

    UPDATE profile SET avatar_id = media_id WHERE id = prof_id;

    INSERT INTO user_profile (uid, user_account_id, profile_id,
                               first_name, last_name, gender, bio,
                               town, birthday_date)
    VALUES (
      gen_random_uuid(), acc_id, prof_id,
      fnames[i], lnames[i], genders[i]::gender_type,
      bios[((i - 1) % 25) + 1],
      towns[i],
      (DATE '1995-01-01' + ((i * 173 + 31) % 3652) * INTERVAL '1 day')::DATE
    );

    prof_ids := prof_ids || prof_id;
  END LOOP;

  -- ── Phase 2: Friendships (50 × 8 = 400) ──────────────────────────────────
  FOR i IN 1..50 LOOP
    FOR j IN 1..8 LOOP
      INSERT INTO friendship (requester_id, addressee_id, status)
      VALUES (prof_ids[i], prof_ids[((i - 1 + j) % 50) + 1], 'accepted')
      ON CONFLICT (requester_id, addressee_id) DO NOTHING;
    END LOOP;
  END LOOP;

  -- ── Phase 3: 10 communities ───────────────────────────────────────────────
  FOR i IN 1..10 LOOP
    INSERT INTO profile (uid) VALUES (gen_random_uuid()) RETURNING id INTO cprof_id;

    INSERT INTO media (uid, media_name, author_id, extension, mime_type, link, size)
    VALUES (
      gen_random_uuid(), 'comm_avatar_' || cunames[i],
      cprof_id, 'jpg', 'image/jpeg',
      comm_covers[i],
      0
    )
    RETURNING id INTO media_id;

    UPDATE profile SET avatar_id = media_id WHERE id = cprof_id;

    INSERT INTO media (uid, media_name, author_id, extension, mime_type, link, size)
    VALUES (
      gen_random_uuid(), 'comm_cover_' || cunames[i],
      cprof_id, 'jpg', 'image/jpeg',
      comm_covers[i],
      0
    )
    RETURNING id INTO cover_id;

    INSERT INTO community (uid, title, bio, community_type, profile_id, username, cover_media_id)
    VALUES (gen_random_uuid(), ctitles[i], cbios[i], 'public', cprof_id, cunames[i], cover_id)
    RETURNING id INTO comm_id;

    cprof_ids := cprof_ids || cprof_id;
    comm_ids  := comm_ids  || comm_id;

    -- 15 members per community, first one is moderator
    FOR j IN 0..14 LOOP
      INSERT INTO community_member (uid, profile_id, community_id, community_role)
      VALUES (
        gen_random_uuid(),
        prof_ids[((i - 1) * 3 + j) % 50 + 1],
        comm_id,
        CASE WHEN j = 0 THEN 'moderator' ELSE 'member' END::community_member_role
      )
      ON CONFLICT (profile_id, community_id) DO NOTHING;
    END LOOP;
  END LOOP;

  -- ── Phase 4: Personal posts with images (50) ─────────────────────────────
  FOR i IN 1..50 LOOP
    INSERT INTO media (uid, media_name, author_id, extension, mime_type, link, size)
    VALUES (
      gen_random_uuid(), 'post_img_' || i,
      prof_ids[i], 'jpg', 'image/jpeg',
      post_imgs[((i - 1) % array_length(post_imgs, 1)) + 1],
      0
    )
    RETURNING id INTO media_id;

    INSERT INTO post (uid, post_text, author_id, is_public_demo)
    VALUES (gen_random_uuid(), ptexts[((i - 1) % 20) + 1], prof_ids[i], FALSE)
    RETURNING id INTO post_id;

    INSERT INTO post_with_media (post_id, media_id, sort_order)
    VALUES (post_id, media_id, 0);

    post_ids     := post_ids     || post_id;
    post_authors := post_authors || prof_ids[i];
  END LOOP;

  -- Text-only posts for users 1-30
  FOR i IN 1..30 LOOP
    INSERT INTO post (uid, post_text, author_id, is_public_demo)
    VALUES (gen_random_uuid(), ptexts[((i + 9) % 20) + 1], prof_ids[i], FALSE)
    RETURNING id INTO post_id;

    post_ids     := post_ids     || post_id;
    post_authors := post_authors || prof_ids[i];
  END LOOP;

  -- Community posts (3 per community = 30)
  FOR i IN 1..10 LOOP
    FOR j IN 1..3 LOOP
      INSERT INTO media (uid, media_name, author_id, extension, mime_type, link, size)
      VALUES (
        gen_random_uuid(),
        'comm_post_' || i || '_' || j,
        cprof_ids[i], 'jpg', 'image/jpeg',
        comm_post_imgs[((i - 1) * 3 + j)],
        0
      )
      RETURNING id INTO media_id;

      INSERT INTO post (uid, post_text, author_id, community_id, is_public_demo)
      VALUES (
        gen_random_uuid(),
        ctexts[((i + j - 2) % 10) + 1],
        cprof_ids[i], comm_ids[i], FALSE
      )
      RETURNING id INTO post_id;

      INSERT INTO post_with_media (post_id, media_id, sort_order)
      VALUES (post_id, media_id, 0);

      post_ids     := post_ids     || post_id;
      post_authors := post_authors || cprof_ids[i];
    END LOOP;
  END LOOP;

  -- ── Phase 5: Likes (≈3 per post) ─────────────────────────────────────────
  FOR i IN 1..array_length(post_ids, 1) LOOP
    FOR j IN 1..3 LOOP
      liker_idx := ((i * 7 + j * 13) % 50) + 1;
      IF prof_ids[liker_idx] <> post_authors[i] THEN
        INSERT INTO like_record (uid, post_id, author_id)
        VALUES (gen_random_uuid(), post_ids[i], prof_ids[liker_idx])
        ON CONFLICT DO NOTHING;
      END IF;
    END LOOP;
  END LOOP;

  -- ── Phase 6: Comments (1 per post) ───────────────────────────────────────
  FOR i IN 1..array_length(post_ids, 1) LOOP
    liker_idx := ((i * 11 + 5) % 50) + 1;
    IF prof_ids[liker_idx] <> post_authors[i] THEN
      INSERT INTO comment (uid, comment_text, post_id, author_id)
      VALUES (gen_random_uuid(), cmts[((i - 1) % 20) + 1], post_ids[i], prof_ids[liker_idx]);
    END IF;
  END LOOP;

  -- ── Phase 7: 40 game questions ────────────────────────────────────────────
  -- slug column was dropped by migration 000024, so it is excluded here
  INSERT INTO game_question (uid, game_type, question_text, correct_answer, answer_unit)
  VALUES
    (gen_random_uuid(),'number_duel','В каком году началась Вторая мировая война?',1939,'год'),
    (gen_random_uuid(),'number_duel','В каком году Юрий Гагарин совершил первый космический полёт?',1961,'год'),
    (gen_random_uuid(),'number_duel','В каком году Колумб достиг берегов Америки?',1492,'год'),
    (gen_random_uuid(),'number_duel','В каком году произошла битва при Ватерлоо?',1815,'год'),
    (gen_random_uuid(),'number_duel','Сколько лет фактически длилась Столетняя война?',116,'лет'),
    (gen_random_uuid(),'number_duel','В каком году была принята Конституция США?',1787,'год'),
    (gen_random_uuid(),'number_duel','В каком году прекратил существование СССР?',1991,'год'),
    (gen_random_uuid(),'number_duel','В каком году Пётр I основал Санкт-Петербург?',1703,'год'),
    (gen_random_uuid(),'number_duel','Скорость света в вакууме (км/с, округлённо)?',299792,'км/с'),
    (gen_random_uuid(),'number_duel','Сколько планет в Солнечной системе?',8,'планет'),
    (gen_random_uuid(),'number_duel','Атомный номер водорода в таблице Менделеева?',1,'номер'),
    (gen_random_uuid(),'number_duel','В каком году был открыт пенициллин?',1928,'год'),
    (gen_random_uuid(),'number_duel','Температура кипения воды при нормальном давлении (°C)?',100,'°C'),
    (gen_random_uuid(),'number_duel','Сколько костей в скелете взрослого человека?',206,'костей'),
    (gen_random_uuid(),'number_duel','Молярная масса углекислого газа (г/моль)?',44,'г/моль'),
    (gen_random_uuid(),'number_duel','Средний диаметр Земли (км)?',12742,'км'),
    (gen_random_uuid(),'number_duel','Сколько простых чисел от 1 до 50 включительно?',15,'чисел'),
    (gen_random_uuid(),'number_duel','Квадратный корень из 144?',12,NULL),
    (gen_random_uuid(),'number_duel','Сколько нулей содержит число один миллиард?',9,'нулей'),
    (gen_random_uuid(),'number_duel','Сколько градусов в полном круге?',360,'градусов'),
    (gen_random_uuid(),'number_duel','Какое седьмое число в последовательности Фибоначчи?',13,NULL),
    (gen_random_uuid(),'number_duel','Длина реки Нил (км)?',6853,'км'),
    (gen_random_uuid(),'number_duel','В каком году был открыт Суэцкий канал?',1869,'год'),
    (gen_random_uuid(),'number_duel','Высота горы Эверест над уровнем моря (метров)?',8849,'метров'),
    (gen_random_uuid(),'number_duel','Сколько официально признанных стран в Африке?',54,'стран'),
    (gen_random_uuid(),'number_duel','Площадь России в миллионах кв. км (округлённо)?',17,'млн км²'),
    (gen_random_uuid(),'number_duel','В каком году официально распались The Beatles?',1970,'год'),
    (gen_random_uuid(),'number_duel','Сколько струн у классической гитары?',6,'струн'),
    (gen_random_uuid(),'number_duel','В каком году написана Лунная соната Бетховена?',1801,'год'),
    (gen_random_uuid(),'number_duel','Сколько нот в одной октаве?',7,'нот'),
    (gen_random_uuid(),'number_duel','Сколько игроков от команды на площадке в баскетболе?',5,'игроков'),
    (gen_random_uuid(),'number_duel','В каком году основан ФК Реал Мадрид?',1902,'год'),
    (gen_random_uuid(),'number_duel','В каком году прошли первые современные Олимпийские игры?',1896,'год'),
    (gen_random_uuid(),'number_duel','Сколько геймов нужно выиграть для победы в сете (минимум)?',6,'геймов'),
    (gen_random_uuid(),'number_duel','Сколько минут длится стандартный футбольный матч?',90,'минут'),
    (gen_random_uuid(),'number_duel','Длина дистанции марафона (км, округлённо)?',42,'км'),
    (gen_random_uuid(),'number_duel','В каком году создан язык программирования Python?',1991,'год'),
    (gen_random_uuid(),'number_duel','Сколько бит в одном байте?',8,'бит'),
    (gen_random_uuid(),'number_duel','В каком году выпущен первый микропроцессор Intel 4004?',1971,'год'),
    (gen_random_uuid(),'number_duel','Количество пикселей по горизонтали в Full HD?',1920,'пикселей');

  -- ── Phase 8: Search outbox ────────────────────────────────────────────────
  FOR i IN 1..50 LOOP
    INSERT INTO search_outbox (entity_type, entity_id, operation)
    VALUES ('user', prof_ids[i], 'upsert');
  END LOOP;

  FOR i IN 1..10 LOOP
    INSERT INTO search_outbox (entity_type, entity_id, operation)
    VALUES ('community', comm_ids[i], 'upsert');
  END LOOP;

  FOR i IN 1..array_length(post_ids, 1) LOOP
    INSERT INTO search_outbox (entity_type, entity_id, operation)
    VALUES ('post', post_ids[i], 'upsert');
  END LOOP;

END $$;
