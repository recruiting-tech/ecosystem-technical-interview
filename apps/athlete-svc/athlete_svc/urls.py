from django.urls import path

from athlete_svc.core.infrastructure.http.views import AthletesView, AthleteDetailView

urlpatterns = [
    path("athletes", AthletesView.as_view(), name="athletes"),
    path("athletes/<uuid:athlete_id>", AthleteDetailView.as_view(), name="athlete-detail"),
]
