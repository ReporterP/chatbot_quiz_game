import { useEffect, useState, useRef, useCallback, useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useDispatch, useSelector } from 'react-redux';
import { DndContext, closestCenter, PointerSensor, TouchSensor, useSensor, useSensors, useDroppable, DragOverlay } from '@dnd-kit/core';
import { SortableContext, verticalListSortingStrategy, useSortable, arrayMove } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import Header from '../components/Header';
import QuestionForm from '../components/QuestionForm';
import { loadQuiz } from '../store/quizSlice';
import { updateQuiz, createCategory, updateCategory as apiUpdateCategory, deleteCategory as apiDeleteCategory, reorderQuiz, createQuestion, updateQuestion, deleteQuestion, addQuestionImage, exportQuiz, importQuiz } from '../api/quizzes';
import './QuizEditPage.css';
import { createSession } from '../api/sessions';

const PRESET_COLORS = ['#e21b3c', '#1368ce', '#d89e00', '#26890c', '#864cbf', '#0aa3b1'];

function SortableItem({ id, children }) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({ id });
  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.4 : 1,
  };
  return (
    <div ref={setNodeRef} style={style} {...attributes}>
      <div className="drag-handle" {...listeners} title="Перетащить">⠿</div>
      {children}
    </div>
  );
}

function CategoryDropZone({ id, children, className }) {
  const { setNodeRef, isOver } = useDroppable({ id });
  return (
    <div ref={setNodeRef} className={`${className}${isOver ? ' drop-active' : ''}`}>
      {children}
    </div>
  );
}

function findContainerIn(containers, id) {
  const s = String(id);
  if (s.startsWith('drop-')) return s.replace('drop-', '');
  if (s.startsWith('cat-')) return s.replace('cat-', '');
  for (const [key, items] of Object.entries(containers)) {
    if (items.includes(s)) return key;
  }
  return null;
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
  const [containers, setContainers] = useState({});
  const [activeId, setActiveId] = useState(null);
  const titleTimer = useRef(null);
  const containersRef = useRef({});

  useEffect(() => { containersRef.current = containers; }, [containers]);

  const toggleCollapse = (catId) => {
    setCollapsedCats((prev) => ({ ...prev, [catId]: !prev[catId] }));
  };

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 200, tolerance: 6 } }),
  );

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
      const c = {};
      (quiz.categories || []).forEach(cat => {
        c[String(cat.id)] = [...(cat.questions || [])]
          .sort((a, b) => a.order_num - b.order_num)
          .map(q => `q-${q.id}`);
      });
      c['orphan'] = [...(quiz.questions || [])]
        .sort((a, b) => a.order_num - b.order_num)
        .map(q => `q-${q.id}`);
      setContainers(c);
    }
  }, [quiz]);

  const reload = useCallback(() => dispatch(loadQuiz(id)), [id]);

  const categories = useMemo(() =>
    [...(quiz?.categories || [])].sort((a, b) => a.order_num - b.order_num), [quiz]);

  const questionsMap = useMemo(() => {
    const map = {};
    (quiz?.categories || []).forEach(cat => {
      (cat.questions || []).forEach(q => { map[`q-${q.id}`] = q; });
    });
    (quiz?.questions || []).forEach(q => { map[`q-${q.id}`] = q; });
    return map;
  }, [quiz]);

  const categoryIds = useMemo(() => categories.map(c => `cat-${c.id}`), [categories]);

  const totalQuestions = useMemo(() =>
    Object.values(containers).reduce((sum, items) => sum + items.length, 0), [containers]);

  const buildPayload = (cs, cats) => ({
    categories: cats.map((c, i) => ({
      id: c.id,
      order_num: i,
      questions: (cs[String(c.id)] || []).map((qId, qi) => ({
        id: Number(String(qId).replace('q-', '')),
        order_num: qi,
      })),
    })),
    orphan_questions: (cs['orphan'] || []).map((qId, qi) => ({
      id: Number(String(qId).replace('q-', '')),
      order_num: qi,
    })),
  });

  // --- DnD handlers ---
  const handleDragStart = useCallback((event) => {
    setActiveId(String(event.active.id));
  }, []);

  const handleDragOver = useCallback((event) => {
    const { active, over } = event;
    if (!over) return;
    const activeStr = String(active.id);
    if (!activeStr.startsWith('q-')) return;

    setContainers(prev => {
      const ac = findContainerIn(prev, activeStr);
      const oc = findContainerIn(prev, String(over.id));
      if (!ac || !oc || ac === oc) return prev;

      const activeItems = [...(prev[ac] || [])];
      const overItems = [...(prev[oc] || [])];
      const activeIndex = activeItems.indexOf(activeStr);
      if (activeIndex === -1) return prev;

      activeItems.splice(activeIndex, 1);
      const overIndex = overItems.indexOf(String(over.id));
      overItems.splice(overIndex >= 0 ? overIndex : overItems.length, 0, activeStr);

      return { ...prev, [ac]: activeItems, [oc]: overItems };
    });
  }, []);

  const handleDragEnd = useCallback(async (event) => {
    const { active, over } = event;
    setActiveId(null);
    if (!over) return;

    const activeStr = String(active.id);
    const overStr = String(over.id);

    if (activeStr.startsWith('cat-') && overStr.startsWith('cat-')) {
      const cats = [...categories];
      const oldIndex = cats.findIndex(c => `cat-${c.id}` === activeStr);
      const newIndex = cats.findIndex(c => `cat-${c.id}` === overStr);
      if (oldIndex === -1 || newIndex === -1 || oldIndex === newIndex) return;

      const reordered = arrayMove(cats, oldIndex, newIndex);
      const payload = buildPayload(containersRef.current, reordered);
      await reorderQuiz(id, payload);
      reload();
      return;
    }

    if (activeStr.startsWith('q-')) {
      let finalContainers;
      setContainers(prev => {
        let next = { ...prev };
        const ac = findContainerIn(next, activeStr);
        const oc = findContainerIn(next, overStr);
        if (ac && oc && ac === oc && activeStr !== overStr) {
          const items = [...next[ac]];
          const oldIdx = items.indexOf(activeStr);
          const newIdx = items.indexOf(overStr);
          if (oldIdx !== -1 && newIdx !== -1 && oldIdx !== newIdx) {
            next = { ...next, [ac]: arrayMove(items, oldIdx, newIdx) };
          }
        }
        finalContainers = next;
        return next;
      });

      const payload = buildPayload(finalContainers, categories);
      await reorderQuiz(id, payload);
      reload();
    }
  }, [categories, id, reload]);

  // --- CRUD handlers ---
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

  const handleUpdateQuestion = async (questionId, data, categoryId) => {
    data.category_id = categoryId ?? null;
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

  if (loading || !quiz) return <><Header /><div className="loading">Загрузка...</div></>;

  const orphanIds = containers['orphan'] || [];

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

        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragStart={handleDragStart}
          onDragOver={handleDragOver}
          onDragEnd={handleDragEnd}
        >
          <SortableContext items={categoryIds} strategy={verticalListSortingStrategy}>
            {categories.map((cat) => {
              const qIds = containers[String(cat.id)] || [];
              const isCollapsed = !!collapsedCats[cat.id];
              return (
                <SortableItem key={cat.id} id={`cat-${cat.id}`}>
                  <CategoryDropZone id={`drop-${cat.id}`} className={`category-section${isCollapsed ? ' collapsed' : ''}`}>
                    <CategoryHeader cat={cat} onRename={handleRenameCat} onDelete={handleDeleteCategory} collapsed={isCollapsed} onToggle={() => toggleCollapse(cat.id)} questionsCount={qIds.length} />
                    {!isCollapsed && (
                      <>
                        <SortableContext items={qIds} strategy={verticalListSortingStrategy}>
                          {qIds.map(qId => {
                            const q = questionsMap[qId];
                            if (!q) return null;
                            return editingId === q.id ? (
                              <QuestionForm key={q.id} initial={q} orderNum={q.order_num} onSave={(data) => handleUpdateQuestion(q.id, data, cat.id)} onCancel={() => setEditingId(null)} />
                            ) : (
                              <SortableItem key={q.id} id={qId}>
                                <QuestionCard q={q} onEdit={() => setEditingId(q.id)} onDelete={() => handleDeleteQuestion(q.id)} />
                              </SortableItem>
                            );
                          })}
                        </SortableContext>
                        {addingQuestionCatId === cat.id ? (
                          <QuestionForm orderNum={qIds.length} onSave={(data) => handleSaveQuestion(cat.id, data)} onCancel={() => setAddingQuestionCatId(null)} />
                        ) : (
                          <button className="btn btn-outline btn-sm" style={{ width: '100%' }} onClick={() => setAddingQuestionCatId(cat.id)}>+ Вопрос</button>
                        )}
                      </>
                    )}
                  </CategoryDropZone>
                </SortableItem>
              );
            })}
          </SortableContext>

          <CategoryDropZone id="drop-orphan" className="category-section orphan-section">
            <div className="category-header">
              <h3 className="cat-title">Без категории</h3>
            </div>
            <SortableContext items={orphanIds} strategy={verticalListSortingStrategy}>
              {orphanIds.map(qId => {
                const q = questionsMap[qId];
                if (!q) return null;
                return editingId === q.id ? (
                  <QuestionForm key={q.id} initial={q} orderNum={q.order_num} onSave={(data) => handleUpdateQuestion(q.id, data, null)} onCancel={() => setEditingId(null)} />
                ) : (
                  <SortableItem key={q.id} id={qId}>
                    <QuestionCard q={q} onEdit={() => setEditingId(q.id)} onDelete={() => handleDeleteQuestion(q.id)} />
                  </SortableItem>
                );
              })}
            </SortableContext>
            {addingQuestionCatId === 'orphan' ? (
              <QuestionForm orderNum={orphanIds.length} onSave={(data) => handleSaveQuestion(null, data)} onCancel={() => setAddingQuestionCatId(null)} />
            ) : (
              <button className="btn btn-outline btn-sm" style={{ width: '100%' }} onClick={() => setAddingQuestionCatId('orphan')}>+ Вопрос</button>
            )}
          </CategoryDropZone>

          <DragOverlay>
            {activeId && activeId.startsWith('q-') && questionsMap[activeId] ? (
              <div className="question-card drag-overlay">
                <h4>{questionsMap[activeId].text}</h4>
              </div>
            ) : activeId && activeId.startsWith('cat-') ? (
              <div className="category-section drag-overlay" style={{ padding: '12px 16px' }}>
                <h3 className="cat-title">{categories.find(c => `cat-${c.id}` === activeId)?.title}</h3>
              </div>
            ) : null}
          </DragOverlay>
        </DndContext>

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
