from django.db import models
from django.db.models import F
from django.utils import timezone


class Quiz(models.Model):
    title = models.CharField(max_length=200)
    description = models.TextField(blank=True)
    is_active = models.BooleanField(default=True)

    def __str__(self):
        return self.title


class Question(models.Model):
    quiz = models.ForeignKey(Quiz, on_delete=models.CASCADE, related_name="questions")
    text = models.TextField()
    order = models.PositiveIntegerField(default=1)
    points = models.PositiveIntegerField(default=1)

    class Meta:
        ordering = ["order", "id"]
        unique_together = ("quiz", "order")

    def __str__(self):
        return f"{self.quiz.title}: {self.text[:50]}"


class AnswerOption(models.Model):
    question = models.ForeignKey(Question, on_delete=models.CASCADE, related_name="options")
    text = models.CharField(max_length=300)
    is_correct = models.BooleanField(default=False)

    def __str__(self):
        return f"{self.question.id}: {self.text[:50]}"


class QuizSession(models.Model):
    class Status(models.TextChoices):
        DRAFT = "draft", "Draft"
        ACTIVE = "active", "Active"
        COMPLETED = "completed", "Completed"

    quiz = models.ForeignKey(Quiz, on_delete=models.CASCADE, related_name="sessions")
    join_code = models.CharField(max_length=10, unique=True)
    status = models.CharField(max_length=12, choices=Status.choices, default=Status.DRAFT)
    current_question = models.ForeignKey(
        Question, on_delete=models.SET_NULL, null=True, blank=True, related_name="+"
    )
    current_question_revealed = models.BooleanField(default=False)
    created_at = models.DateTimeField(auto_now_add=True)
    started_at = models.DateTimeField(null=True, blank=True)
    ended_at = models.DateTimeField(null=True, blank=True)

    def __str__(self):
        return f"{self.quiz.title} ({self.join_code})"

    def next_question(self):
        if self.current_question_id is None:
            return self.quiz.questions.order_by("order", "id").first()
        return (
            self.quiz.questions.filter(order__gt=self.current_question.order)
            .order_by("order", "id")
            .first()
        )

    def active_participants(self):
        return self.participants.filter(is_active=True)

    def all_answered_current_question(self):
        if not self.current_question_id:
            return False
        participants_count = self.active_participants().count()
        if participants_count == 0:
            return False
        answers_count = ParticipantAnswer.objects.filter(
            participant__session=self,
            participant__is_active=True,
            question_id=self.current_question_id,
        ).count()
        return answers_count >= participants_count


class Participant(models.Model):
    session = models.ForeignKey(
        QuizSession, on_delete=models.CASCADE, related_name="participants"
    )
    telegram_user_id = models.BigIntegerField()
    username = models.CharField(max_length=150, blank=True)
    first_name = models.CharField(max_length=150, blank=True)
    last_name = models.CharField(max_length=150, blank=True)
    joined_at = models.DateTimeField(auto_now_add=True)
    score = models.IntegerField(default=0)
    is_active = models.BooleanField(default=True)

    class Meta:
        unique_together = ("session", "telegram_user_id")

    def __str__(self):
        return f"{self.telegram_user_id} ({self.session.join_code})"


class ParticipantAnswer(models.Model):
    participant = models.ForeignKey(
        Participant, on_delete=models.CASCADE, related_name="answers"
    )
    question = models.ForeignKey(Question, on_delete=models.CASCADE, related_name="answers")
    selected_option = models.ForeignKey(
        AnswerOption, on_delete=models.CASCADE, related_name="+"
    )
    is_correct = models.BooleanField(default=False)
    answered_at = models.DateTimeField(default=timezone.now)

    class Meta:
        unique_together = ("participant", "question")

    def save(self, *args, **kwargs):
        is_new = self._state.adding
        self.is_correct = self.selected_option.is_correct
        super().save(*args, **kwargs)
        if is_new and self.is_correct:
            Participant.objects.filter(pk=self.participant_id).update(
                score=F("score") + self.question.points
            )

