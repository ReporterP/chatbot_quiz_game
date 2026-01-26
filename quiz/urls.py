from django.urls import path

from . import views


urlpatterns = [
    path("", views.home, name="home"),
    path("sessions/<int:session_id>/", views.session_detail, name="session_detail"),
    path("screen/<str:join_code>/", views.session_screen, name="session_screen"),
]

