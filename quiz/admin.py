from django.contrib import admin

from .models import (
    AnswerOption,
    Participant,
    ParticipantAnswer,
    Question,
    Quiz,
    QuizSession,
)


class AnswerOptionInline(admin.TabularInline):
    model = AnswerOption
    extra = 2


class QuestionInline(admin.TabularInline):
    model = Question
    extra = 1


@admin.register(Quiz)
class QuizAdmin(admin.ModelAdmin):
    list_display = ("title", "is_active")
    search_fields = ("title",)
    inlines = [QuestionInline]


@admin.register(Question)
class QuestionAdmin(admin.ModelAdmin):
    list_display = ("quiz", "order", "text", "points")
    list_filter = ("quiz",)
    inlines = [AnswerOptionInline]


@admin.register(AnswerOption)
class AnswerOptionAdmin(admin.ModelAdmin):
    list_display = ("question", "text", "is_correct")
    list_filter = ("question", "is_correct")


@admin.register(QuizSession)
class QuizSessionAdmin(admin.ModelAdmin):
    list_display = ("quiz", "join_code", "status", "current_question")
    list_filter = ("status", "quiz")
    search_fields = ("join_code",)


@admin.register(Participant)
class ParticipantAdmin(admin.ModelAdmin):
    list_display = ("session", "telegram_user_id", "username", "score", "is_active")
    list_filter = ("session", "is_active")
    search_fields = ("telegram_user_id", "username")


@admin.register(ParticipantAnswer)
class ParticipantAnswerAdmin(admin.ModelAdmin):
    list_display = ("participant", "question", "selected_option", "is_correct")
    list_filter = ("question", "is_correct")

