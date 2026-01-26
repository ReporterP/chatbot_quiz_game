import json
import logging
import os
import random
import string
import urllib.error
import urllib.parse
import urllib.request

from django.contrib.admin.views.decorators import staff_member_required
from django.db import transaction
from django.shortcuts import get_object_or_404, redirect, render
from django.utils import timezone

from .models import Quiz, QuizSession

logger = logging.getLogger(__name__)


def _generate_join_code():
    return "".join(random.choices(string.digits, k=6))


def _send_telegram_question(session: QuizSession):
    token = os.getenv("BOT_TOKEN")
    if not token:
        logger.warning("BOT_TOKEN is not set; skipping Telegram notify")
        return
    if not session.current_question:
        return

    options = list(session.current_question.options.order_by("id").values("id", "text"))
    if not options:
        return

    keyboard = [
        [
            {
                "text": option["text"],
                "callback_data": f"answer:{session.current_question.id}:{option['id']}",
            }
        ]
        for option in options
    ]
    payload = {
        "text": f"Новый вопрос:\n{session.current_question.text}",
        "reply_markup": json.dumps({"inline_keyboard": keyboard}, ensure_ascii=False),
    }

    for participant in session.participants.filter(is_active=True).only("telegram_user_id"):
        data = {**payload, "chat_id": participant.telegram_user_id}
        request = urllib.request.Request(
            url=f"https://api.telegram.org/bot{token}/sendMessage",
            data=urllib.parse.urlencode(data).encode("utf-8"),
        )
        try:
            with urllib.request.urlopen(request, timeout=5):
                pass
        except urllib.error.URLError as exc:
            logger.warning("Failed to notify %s: %s", participant.telegram_user_id, exc)


@staff_member_required
def home(request):
    if request.method == "POST":
        quiz_id = request.POST.get("quiz_id")
        quiz = get_object_or_404(Quiz, pk=quiz_id, is_active=True)
        join_code = _generate_join_code()
        while QuizSession.objects.filter(join_code=join_code).exists():
            join_code = _generate_join_code()
        QuizSession.objects.create(quiz=quiz, join_code=join_code)
        return redirect("home")

    sessions = QuizSession.objects.select_related("quiz", "current_question").order_by(
        "-created_at"
    )
    quizzes = Quiz.objects.filter(is_active=True).order_by("title")
    return render(
        request,
        "quiz/home.html",
        {"sessions": sessions, "quizzes": quizzes},
    )


@staff_member_required
def session_detail(request, session_id):
    session = get_object_or_404(
        QuizSession.objects.select_related("quiz", "current_question"), pk=session_id
    )

    if request.method == "POST":
        action = request.POST.get("action")
        if action == "start":
            session.status = QuizSession.Status.ACTIVE
            session.started_at = session.started_at or timezone.now()
            session.current_question_revealed = False
            session.save()
        elif action == "next":
            next_question = session.next_question()
            if next_question:
                session.current_question = next_question
                session.current_question_revealed = False
            else:
                session.status = QuizSession.Status.COMPLETED
                session.ended_at = timezone.now()
            session.save()
            if next_question:
                _send_telegram_question(session)
        elif action == "reveal":
            session.current_question_revealed = True
            session.save(update_fields=["current_question_revealed"])
        elif action == "end":
            session.status = QuizSession.Status.COMPLETED
            session.ended_at = timezone.now()
            session.save()
        return redirect("session_detail", session_id=session.id)

    if (
        session.status == QuizSession.Status.ACTIVE
        and session.current_question_id
        and not session.current_question_revealed
        and session.all_answered_current_question()
    ):
        session.current_question_revealed = True
        session.save(update_fields=["current_question_revealed"])

    participants = session.participants.order_by("-score", "joined_at")
    total_questions = session.quiz.questions.count()
    return render(
        request,
        "quiz/session_detail.html",
        {
            "session": session,
            "participants": participants,
            "total_questions": total_questions,
        },
    )


def session_screen(request, join_code):
    session = get_object_or_404(
        QuizSession.objects.select_related("quiz", "current_question"),
        join_code=join_code,
    )
    if (
        session.status == QuizSession.Status.ACTIVE
        and session.current_question_id
        and not session.current_question_revealed
        and session.all_answered_current_question()
    ):
        with transaction.atomic():
            session.current_question_revealed = True
            session.save(update_fields=["current_question_revealed"])

    participants = session.participants.order_by("-score", "joined_at")
    return render(
        request,
        "quiz/session_screen.html",
        {"session": session, "participants": participants},
    )

