# user
- `id` - униальный id пользователя
- `username` - юзернейм
- `email` - почта
- `phone` - номер телефона
- `password_hash` - хэш пароля
- `createdAt` - дата создания
- `updatedAt` - дата обновления
- `isActive` - активный ли аккаунт
# profile
- `id` - униальный id профиля
- `firstName` - имя
- `lastName` - фамилия
- `bio` - описание профиля
- `birthdayDate` - дата рождения
- `gender_id` - пол
- `user_id` - пользователь, которому принадлежит профиль
- `avatar` - аватарка профиля
- `createdAt` - дата создания
- `updatedAt` - дата обновления
- `isActive` - активный ли аккаунт

# gender
- `id` - униальный id пола
- `genderName` - название пола

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
- `id` - униальный id поста
- `text` - текст поста
- `createdAt` - дата создания
- `updatedAt` - дата обновления
- `isActive` - активный ли пост
# postByProfile
- `post_id` - id поста
- `profile_id` - профиль, отправивший пост
# postByGroup
- `post_id` - id поста
- `group_id` - сообщество, отправившее пост
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
# chatMemberProfile
- `id` - униальный id участника чата
- `profile_id` - профиль-участник чата
- `chat_id` - чат
- `joinedAt` - дата вступления в чат
- `leaveAt` - дата выхода из чата
- `role_id` - роль в чате
# chatMemberGroup
- `id` - униальный id участника чата
- `group_id` - группа-участник чата
- `chat_id` - чат
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
- `id` - униальный id сообщения
- `text` - текст сообщения
- `parentMessage_id` - id родительского сообщения (для ответов)
- `chat_id` - чат
- `status_id` - статус сообщения
- `sticker_id` - сообщение = стикер
- `createdAt` - дата создания
- `updatedAt` - дата изменения
- `isDeleted` - удалено ли сообщение
# messageByProfile
- `message_id` - id сообщения
- `profile_id` - профиль, отправивший сообщение
# messageByGroup
- `message_id` - id сообщения
- `group_id` - группа, отправившая сообщение
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
- `owner_id` - профиль владельца группы (profile table)
- `avatar_id` - аватарка группы (media table)
- `createdAt` - дата создания
- `updatedAt` - дата обновления
- `isDeleted` - удалена ли группа
# groupMemberProfile
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
- `id` - униальный id комментария
- `text` - текст комментария
- `post_id` - пост, под которым оставлен комментарий
- `parentComment_id` - родительский комментарий(для ответов)
- `sticker_id` - комментарий = стикер
- `createdAt` - дата создания
- `updatedAt` - дата обновления
- `isDeleted` - удален ли комментарий
# commentByProfile
- `comment` - id комментария
- `profile_id` - профиль, который оставил сообщение
# commentByGroup
- `comment` - id комментария
- `group_id` - группа, которая оставила комментарий
# commentWithMedia
- `comment_id` - комментарий с медиа
- `media_id` - медиафайл комментария
- `order` - порядковый номер медиафайла в комментарии

# likeByProfile
- `id` - уникальный id лайка
- `profile_id` - профиль, поставивший лайк
- `createdAt` - дата создания
# likeByGroup
- `id` - уникальный id лайка
- `group_id` - группа, которая поставила лайк
- `createdAt` - дата создания
# likeToPost
- `id` - уникальный id лайка
- `post_id` - пост, под которым поставлен лайк
- `createdAt` - дата создания
# likeToComment
- `id` - уникальный id лайка
- `comment_id` - комментарий, которому поставлен лайк
- `createdAt` - дата создания

# reaction
- `id` - униальный id реакции
- `message_id` - сообщение, к которому приложена реакция
- `type_id` - тип реакции
- `createdAt` - дата создания реакции
# reactionByProfile
- `reaction_id` - id реакции
- `profile_id` - профиль, оставивший реакцию
# reactionByGroup
- `reaction_id` - id реакции
- `group_id` - группа, оставившая реакцию
# reactionType
- `id` - униальный id типа реакции
- `typeName` - название типа реакции

# friendship
- `friend1_id` - профиль одного друга
- `friend2_id` - профиль другого друга
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
# sticker
- `link` - ссылка на стикер
- `size` - размер стикера
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
- `id` - униальный id рекламы
- `title` - название рекламы
- `description` - описание рекламы
- `link` - ссылка на рекламируемый продукт или услугу
- `media_id` - медиа контент рекламы
- `createdAt` - дата создания
- `updatedAt` - дата обновления
- `isDeleted` - удален ли рекламный пост
# adByProfile
- `ad_id` - id рекламы
- `profile_id` - профиль, который создал рекламу
# adByGroup
- `ad_id` - id рекламы
- `group_id` - группа, которая создала рекламу
# adMeta
- `id` - униальный id метаданных рекламы
- `ad_id` - реклама, к которой относятся метаданные
- `key` - ключ метаданных
- `value` - значение метаданных