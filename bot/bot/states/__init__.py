from aiogram.fsm.state import State, StatesGroup


class QuizStates(StatesGroup):
    enter_code = State()
    enter_nickname = State()
    change_nickname = State()
    in_session = State()
