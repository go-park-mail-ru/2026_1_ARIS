# user
- `id` - униальный id пользователя
- `username` - юзернейм
- `email` - почта
- `phone` - номер телефона
- `password_hash` - хэш пароля
- `updatedAt` - дата обновления
# userProfile
- `user_id` - пользователь, которому принадлежит профиль
- `firstName` - имяd
- `lastName` - фамилия
- `bio` - описание профиля
- `birthdayDate` - дата рождения
- `gender` - пол (`int` enum)

# profile
- `id` - уникальный id абстрактного профиля, единой точки для действий и контента
- `avatar_id` - общая аватарка, отображаемая везде
- `createdAt` - дата создания
- `updatedAt` - дата обновления
- `isActive` - активный ли аккаунт

# profileUser
- `abstract_profile_id` - ссылка на `profile`
- `profile_id` - профиль из таблицы `profile`

# profileGroup
- `abstract_profile_id` - ссылка на `profile`
- `group_id` - группа из таблицы `group`

# media
- `id` - униальный id файла
- `name` - называния файла
- `description` - описание файла
- `link` - ссылка на файл
- `mimeType` - MIME тип
- `size` - размер в байтах
- `createdAt` - дата создания
- `isDeleted` - удалено ли

# post
- `id` - уникальный id поста
- `text` - текст поста
- `author_id` - абстрактный профиль, создавший пост (ссылка на `profile`)
- `createdAt` - дата создания
- `updatedAt` - дата обновления
- `isActive` - активный ли пост

# postWithMedia
- `post_id` - пост, к которому прикреплён медиафайл
- `media_id` - медиафайл поста
- `order` - порядковый номер медиафайла в посте

# chat
- `id` - униальный id чата
- `type_id` - тип чата
- `title` - название чата
- `avatar_id` - аватарка чата (media table)
- `createdAt` - дата создания
- `updatedAt` - дата обновления
- `isDeleted` - удален ли чат
# chatMember
- `id` - уникальный id участника чата
- `chat_id` - чат
- `abstract_profile_id` - ссылка на `profile` (профиль или группа)
- `joinedAt` - дата вступления в чат
- `leaveAt` - дата выхода из чата
- `role_id` - роль в чате
# chatRole
- `id` - униальный id роли чата
- `roleName` - название роли
# chatType
- `id` - униальный id типа чата
- `typeName` - название типа чата

# message
- `id` - уникальный id сообщения
- `text` - текст сообщения
- `parentMessage_id` - id родительского сообщения (для ответов)
- `chat_id` - чат
- `status_id` - статус сообщения
- `sticker_id` - сообщение = стикер
- `author_id` - отправитель сообщения (ссылка на `profile`)
- `createdAt` - дата создания
- `updatedAt` - дата изменения
- `isDeleted` - удалено ли сообщение
# messageStatus
- `id` - униальный id статуса сообщения
- `statusName` - название статуса
# messageWithMedia
- `message_id` - сообщение, к которому прикреплён медиафайл
- `media_id` - медиафайл сообщения
- `order` - порядковый номер медиафайла в сообщении

# group
- `id` - униальный id группы
- `title` - название группы
- `description` - описание группы
- `type_id` - тип группы (публичная, приватная)
- `owner_id` - профиль владельца группы (abstructProfile table)
# groupMember
- `id` - униальный id участника группы
- `profile_id` - профиль участника группы
- `group_id` - группа
- `joinedAt` - дата вступления в группу
- `leaveAt` - дата покидания группы
- `role_id` - роль в группе (админ, участник и т.д.)
# groupRole
- `id` - униальный id роли группы
- `roleName` - название роли
# groupType
- `id` - униальный id типа группы
- `typeName` - название типа группы

# comment
- `id` - уникальный id комментария
- `text` - текст комментария
- `post_id` - пост, под которым оставлен комментарий
- `parentComment_id` - родительский комментарий (для ответов)
- `sticker_id` - комментарий = стикер
- `author_id` - автор комментария (ссылка на `profile`)
- `createdAt` - дата создания
- `updatedAt` - дата обновления
- `isDeleted` - удален ли комментарий
# commentWithMedia
- `comment_id` - комментарий с медиа
- `media_id` - медиафайл комментария
- `order` - порядковый номер медиафайла в комментарии

# like
- `id` - уникальный id лайка
- `author_id` - кто поставил лайк (ссылка на `profile`)
- `createdAt` - дата создания
# likeToPost
- `like_id` - уникальный id лайка
- `post_id` - пост, под которым поставлен лайк
# likeToComment
- `like_id` - уникальный id лайка
- `comment_id` - комментарий, которому поставлен лайк

# reaction
- `id` - уникальный id реакции
- `message_id` - сообщение, к которому приложена реакция
- `type_id` - тип реакции
- `author_id` - кто оставил реакцию (ссылка на `profile`)
- `createdAt` - дата создания реакции
# reactionType
- `id` - униальный id типа реакции
- `typeName` - название типа реакции

# friendship
- `friend1_id` - первый участник (ссылка на `profile`)
- `friend2_id` - второй участник (ссылка на `profile`)
- `status_id` - статус дружбы (запрос, принята, отклонена)
- `createdAt` - дата создания
- `updatedAt` - дата обновления
# friendshipStatus
- `id` - униальный id статуса дружбы
- `statusName` - название статуса дружбы

# stickerpack
- `id` - униальный id стикерпака
- `title` - название стикерпака
- `createdAt` - дата создания
- `updatedAt` - дата обновления
- `isDeleted` - удален ли стикерпак
- `author_id` - кто создал стикерпак (ссылка на `profile`) (опционально)
# sticker
- `id` - униальный id стикера
- `link` - ссылка на стикер
- `size` - размер стикера в байтах
- `indexOrder` - порядковый номер стикера в паке
- `pack_id` - пак, которому пренадлежит стикер
- `createdAt` - дата создания
- `isDeleted` - удален ли стикер

# session
- `id` - униальный id сессии
- `profile_id` - id пользователя, которому принадлежит сессия
- `ip` - IP-адрес, с которого была создана сессия
- `userAgent` - User-Agent, с которого была создана сессия
- `createdAt` - дата создания
- `expiredAt` - дата истечения срока действия

# ad
- `id` - уникальный id рекламы
- `title` - название рекламы
- `description` - описание рекламы
- `link` - ссылка на рекламируемый продукт или услугу
- `media_id` - медиа контент рекламы
- `author_id` - кто создал рекламу (ссылка на `profile`)
- `createdAt` - дата создания
- `updatedAt` - дата обновления
- `isDeleted` - удален ли рекламный пост
# adMeta
- `ad_id` - реклама, к которой относятся метаданные
- `key` - ключ метаданных
- `value` - значение метаданных