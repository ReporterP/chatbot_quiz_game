import { useEffect, useState, useRef, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useDispatch, useSelector } from 'react-redux';
import { DndContext, closestCenter, PointerSensor, useSensor, useSensors } from '@dnd-kit/core';
import { SortableContext, verticalListSortingStrategy, useSortable, arrayMove } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import Header from '../components/Header';
import QuestionForm from '../components/QuestionForm';
import { loadQuiz } from '../store/quizSlice';
import { updateQuiz, createCategory, updateCategory as apiUpdateCategory, deleteCategory as apiDeleteCategory, reorderQuiz, createQuestion, updateQuestion, deleteQuestion, addQuestionImage, exportQuiz, importQuiz } from '../api/quizzes';
import { createSession } from '../api/sessions';

const PRESET_COLORS = ['#e21b3c', '#1368ce', '#d89e00', '#26890c', '#864cbf', '#0aa3b1'];

function randomColor() {
  return PRESET_COLORS[Math.floor(Math.random() * PRESET_COLORS.length)];
}

function SortableItem({ id, children }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id });
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };
  return (
    <div ref={setNodeRef} style={style} {...attributes}>
      <div className="drag-handle" {...listeners} title="Перетащить">⠿</div>
      {children}
    </div>
  );
}

export default function QuizEditPage() {
  const { id } = useParams();
  const dispatch = useDispatch();
  const navigate = useNavigate();
  const quiz = useSelector((s) => s.quiz.current);
  const loading = useSelector((s) => s.quiz.loading);

  const [title, setTitle] = useState('');
  const [addingCat, setAddingCat] = useState(false);
  const [newCatTitle, setNewCatTitle] = useState('');
  const [addingQuestionCatId, setAddingQuestionCatId] = useState(null);
  const [editingId, setEditingId] = useState(null);
  const [collapsedCats, setCollapsedCats] = useState({});
  const titleTimer = useRef(null);

  const toggleCollapse = (catId) => {
    setCollapsedCats((prev) => ({ ...prev, [catId]: !prev[catId] }));
  };

  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }));

  useEffect(() => { dispatch(loadQuiz(id)); }, [id]);
  useEffect(() => {
    if (quiz) {
      setTitle(quiz.title);
      setCollapsedCats((prev) => {
        if (Object.keys(prev).length > 0) return prev;
        const map = {};
        (quiz.categories || []).forEach((c) => { map[c.id] = true; });
        return map;
      });
    }
  }, [quiz]);

  const reload = useCallback(() => dispatch(loadQuiz(id)), [id]);

  const handleTitleChange = (val) => {
    setTitle(val);
    clearTimeout(titleTimer.current);
    titleTimer.current = setTimeout(() => {
      if (val.trim()) updateQuiz(id, val.trim());
    }, 600);
  };

  const handleCreateCategory = async (e) => {
    e.preventDefault();
    if (!newCatTitle.trim()) return;
    await createCategory(id, newCatTitle.trim());
    setNewCatTitle('');
    setAddingCat(false);
    reload();
  };

  const handleDeleteCategory = async (catId) => {
    if (!confirm('Удалить категорию и все её вопросы?')) return;
    await apiDeleteCategory(catId);
    reload();
  };

  const handleRenameCat = async (catId, newTitle) => {
    if (!newTitle.trim()) return;
    await apiUpdateCategory(catId, newTitle.trim());
    reload();
  };

  const handleSaveQuestion = async (catId, data) => {
    if (catId) data.category_id = catId;
    const { data: createdQuestion } = await createQuestion(id, data);
    if (data._images?.length && createdQuestion?.id) {
      for (const url of data._images) {
        await addQuestionImage(createdQuestion.id, url);
      }
    }
    setAddingQuestionCatId(null);
    reload();
  };

  const handleUpdateQuestion = async (questionId, data) => {
    await updateQuestion(questionId, data);
    setEditingId(null);
    reload();
  };

  const handleDeleteQuestion = async (questionId) => {
    if (!confirm('Удалить вопрос?')) return;
    await deleteQuestion(questionId);
    reload();
  };

  const handleLaunch = async () => {
    try {
      const { data } = await createSession(Number(id));
      const sessionId = data.session?.id || data.id;
      navigate(`/session/${sessionId}`);
    } catch (err) {
      alert(err.response?.data?.error || 'Ошибка запуска');
    }
  };

  const handleExport = async (format) => {
    try {
      const { data } = await exportQuiz(id, format);
      const blob = new Blob([data], { type: format === 'csv' ? 'text/csv' : 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `${quiz?.title || 'quiz'}.${format}`;
      a.click();
      URL.revokeObjectURL(url);
    } catch { alert('Ошибка экспорта'); }
  };

  const handleImport = async (e) => {
    const file = e.target.files[0];
    if (!file) return;
    try {
      const { data } = await importQuiz(id, file);
      alert(`Импортировано вопросов: ${data.imported_questions}`);
      reload();
    } catch (err) {
      alert(err.response?.data?.error || 'Ошибка импорта');
    }
    e.target.value = '';
  };

  const handleCatDragEnd = async (event) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const cats = [...(quiz.categories || [])].sort((a, b) => a.order_num - b.order_num);
    const oldIndex = cats.findIndex((c) => `cat-${c.id}` === active.id);
    const newIndex = cats.findIndex((c) => `cat-${c.id}` === over.id);
    if (oldIndex === -1 || newIndex === -1) return;

    const reordered = arrayMove(cats, oldIndex, newIndex);
    const payload = {
      categories: reordered.map((c, i) => ({
        id: c.id,
        order_num: i,
        questions: (c.questions || []).map((q, qi) => ({ id: q.id, order_num: qi })),
      })),
    };
    await reorderQuiz(id, payload);
    reload();
  };

  const handleQuestionDragEnd = async (catId, event) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;

    const cat = (quiz.categories || []).find((c) => c.id === catId);
    if (!cat) return;

    const questions = [...(cat.questions || [])].sort((a, b) => a.order_num - b.order_num);
    const oldIndex = questions.findIndex((q) => `q-${q.id}` === active.id);
    const newIndex = questions.findIndex((q) => `q-${q.id}` === over.id);
    if (oldIndex === -1 || newIndex === -1) return;

    const reordered = arrayMove(questions, oldIndex, newIndex);
    const cats = (quiz.categories || []).sort((a, b) => a.order_num - b.order_num);
    const payload = {
      categories: cats.map((c) => ({
        id: c.id,
        order_num: c.order_num,
        questions: c.id === catId
          ? reordered.map((q, i) => ({ id: q.id, order_num: i }))
          : (c.questions || []).map((q, i) => ({ id: q.id, order_num: i })),
      })),
    };
    await reorderQuiz(id, payload);
    reload();
  };

  if (loading || !quiz) return <><Header /><div className="loading">Загрузка...</div></>;

  const categories = [...(quiz.categories || [])].sort((a, b) => a.order_num - b.order_num);
  const orphanQuestions = [...(quiz.questions || [])].sort((a, b) => a.order_num - b.order_num);
  const totalQuestions = categories.reduce((sum, c) => sum + (c.questions?.length || 0), 0) + orphanQuestions.length;

  return (
    <>
      <Header />
      <div className="quiz-edit">
        <div className="quiz-edit-header">
          <button className="btn btn-outline btn-sm" onClick={() => navigate('/dashboard')}>← Назад</button>
          <input className="quiz-title-input" value={title} onChange={(e) => handleTitleChange(e.target.value)} placeholder="Название квиза" />
          <button className="btn btn-success btn-sm" onClick={handleLaunch} disabled={totalQuestions === 0}>Запустить</button>
        </div>
        <div style={{ display: 'flex', gap: 8, marginBottom: 20, flexWrap: 'wrap' }}>
          <button className="btn btn-outline btn-sm" onClick={() => handleExport('json')}>Экспорт JSON</button>
          <button className="btn btn-outline btn-sm" onClick={() => handleExport('csv')}>Экспорт CSV</button>
          <label className="btn btn-outline btn-sm" style={{ cursor: 'pointer' }}>
            Импорт
            <input type="file" accept=".json,.csv" onChange={handleImport} style={{ display: 'none' }} />
          </label>
        </div>

        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleCatDragEnd}>
          <SortableContext items={categories.map((c) => `cat-${c.id}`)} strategy={verticalListSortingStrategy}>
            {categories.map((cat) => {
              const questions = [...(cat.questions || [])].sort((a, b) => a.order_num - b.order_num);
              const isCollapsed = !!collapsedCats[cat.id];
              return (
                <SortableItem key={cat.id} id={`cat-${cat.id}`}>
                  <div className={`category-section${isCollapsed ? ' collapsed' : ''}`}>
                    <CategoryHeader cat={cat} onRename={handleRenameCat} onDelete={handleDeleteCategory} collapsed={isCollapsed} onToggle={() => toggleCollapse(cat.id)} questionsCount={questions.length} />

                    {!isCollapsed && (
                      <>
                        <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={(e) => handleQuestionDragEnd(cat.id, e)}>
                          <SortableContext items={questions.map((q) => `q-${q.id}`)} strategy={verticalListSortingStrategy}>
                            {questions.map((q) =>
                              editingId === q.id ? (
                                <QuestionForm key={q.id} initial={q} orderNum={q.order_num} onSave={(data) => handleUpdateQuestion(q.id, data)} onCancel={() => setEditingId(null)} />
                              ) : (
                                <SortableItem key={q.id} id={`q-${q.id}`}>
                                  <QuestionCard q={q} onEdit={() => setEditingId(q.id)} onDelete={() => handleDeleteQuestion(q.id)} />
                                </SortableItem>
                              )
                            )}
                          </SortableContext>
                        </DndContext>

                        {addingQuestionCatId === cat.id ? (
                          <QuestionForm orderNum={questions.length} onSave={(data) => handleSaveQuestion(cat.id, data)} onCancel={() => setAddingQuestionCatId(null)} />
                        ) : (
                          <button className="btn btn-outline btn-sm" style={{ width: '100%' }} onClick={() => setAddingQuestionCatId(cat.id)}>+ Вопрос</button>
                        )}
                      </>
                    )}
                  </div>
                </SortableItem>
              );
            })}
          </SortableContext>
        </DndContext>

        {orphanQuestions.length > 0 && (
          <div className="category-section" style={{ borderLeft: '4px solid #fdcb6e' }}>
            <div className="category-header">
              <h3 className="cat-title">Без категории</h3>
            </div>
            {orphanQuestions.map((q) =>
              editingId === q.id ? (
                <QuestionForm key={q.id} initial={q} orderNum={q.order_num} onSave={(data) => handleUpdateQuestion(q.id, data)} onCancel={() => setEditingId(null)} />
              ) : (
                <QuestionCard key={q.id} q={q} onEdit={() => setEditingId(q.id)} onDelete={() => handleDeleteQuestion(q.id)} />
              )
            )}
            {addingQuestionCatId === 'orphan' ? (
              <QuestionForm orderNum={orphanQuestions.length} onSave={(data) => handleSaveQuestion(null, data)} onCancel={() => setAddingQuestionCatId(null)} />
            ) : (
              <button className="btn btn-outline btn-sm" style={{ width: '100%' }} onClick={() => setAddingQuestionCatId('orphan')}>+ Вопрос (без категории)</button>
            )}
          </div>
        )}

        {addingCat ? (
          <form onSubmit={handleCreateCategory} style={{ display: 'flex', gap: 12, marginTop: 16 }}>
            <input className="quiz-title-input" value={newCatTitle} onChange={(e) => setNewCatTitle(e.target.value)} placeholder="Название категории..." autoFocus />
            <button type="submit" className="btn btn-success btn-sm">Создать</button>
            <button type="button" className="btn btn-outline btn-sm" onClick={() => setAddingCat(false)}>Отмена</button>
          </form>
        ) : (
          <button className="btn btn-primary" style={{ width: '100%', marginTop: 16 }} onClick={() => setAddingCat(true)}>+ Добавить категорию</button>
        )}
      </div>
    </>
  );
}

function CategoryHeader({ cat, onRename, onDelete, collapsed, onToggle, questionsCount }) {
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState(cat.title);

  const save = () => {
    if (title.trim() && title.trim() !== cat.title) onRename(cat.id, title);
    setEditing(false);
  };

  return (
    <div className="category-header">
      <div className="cat-title-row">
        <button className={`collapse-toggle${collapsed ? ' collapsed' : ''}`} onClick={onToggle} title={collapsed ? 'Развернуть' : 'Свернуть'}>▶</button>
        {editing ? (
          <input className="cat-title-input" value={title} onChange={(e) => setTitle(e.target.value)} onBlur={save} onKeyDown={(e) => e.key === 'Enter' && save()} autoFocus />
        ) : (
          <h3 className="cat-title" onDoubleClick={() => setEditing(true)}>{cat.title}</h3>
        )}
        {collapsed && <span className="cat-questions-badge">{questionsCount} вопр.</span>}
      </div>
      <div style={{ display: 'flex', gap: 6 }}>
        <button className="btn btn-outline btn-sm" onClick={() => setEditing(true)}>✎</button>
        <button className="btn btn-danger btn-sm" onClick={() => onDelete(cat.id)}>✕</button>
      </div>
    </div>
  );
}

function QuestionCard({ q, onEdit, onDelete }) {
  return (
    <div className="question-card">
      <div className="question-card-header">
        <h4>{q.text}</h4>
        <div style={{ display: 'flex', gap: 6 }}>
          <button className="btn btn-outline btn-sm" onClick={onEdit}>Изменить</button>
          <button className="btn btn-danger btn-sm" onClick={onDelete}>Удалить</button>
        </div>
      </div>
      {q.images?.length > 0 && (
        <div className="question-images-preview">
          {q.images.map((img) => (
            <img key={img.id} src={img.url} alt="" className="q-thumb" />
          ))}
        </div>
      )}
      <div className="question-options">
        {q.options?.map((o) => (
          <div key={o.id} className={`question-option ${o.is_correct ? 'correct' : ''}`} style={o.color ? { borderLeftColor: o.color } : {}}>
            <span className="dot" style={o.color ? { background: o.color } : {}} />
            {o.text}
            {o.is_correct && ' ✓'}
          </div>
        ))}
      </div>
    </div>
  );
}
