import { useState } from 'react';
import { uploadImage, addQuestionImage, deleteQuestionImage } from '../api/quizzes';

const PRESET_COLORS = ['#e21b3c', '#1368ce', '#d89e00', '#26890c', '#864cbf', '#0aa3b1'];
const defaultOption = (i) => ({ text: '', is_correct: false, color: PRESET_COLORS[i % PRESET_COLORS.length] });

const QUESTION_TYPES = [
  { value: 'single_choice', label: 'Один ответ' },
  { value: 'multiple_choice', label: 'Несколько ответов' },
  { value: 'ordering', label: 'Сортировка', webOnly: true },
  { value: 'matching', label: 'Соотнесение', webOnly: true },
  { value: 'numeric', label: 'Числовой ответ' },
];

export default function QuestionForm({ initial, orderNum, onSave, onCancel, quizMode }) {
  const initType = initial?.type || 'single_choice';
  const [type, setType] = useState(initType);
  const [text, setText] = useState(initial?.text || '');
  const [options, setOptions] = useState(() => {
    if (initial?.options?.length) {
      return initial.options.map((o) => ({
        text: o.text,
        is_correct: o.is_correct,
        color: o.color || PRESET_COLORS[0],
        correct_position: o.correct_position ?? null,
        match_text: o.match_text || '',
      }));
    }
    if (initType === 'numeric') return [];
    if (initType === 'ordering') return [{ text: '', correct_position: 1 }, { text: '', correct_position: 2 }];
    if (initType === 'matching') return [{ text: '', match_text: '' }, { text: '', match_text: '' }];
    return [{ text: '', is_correct: true, color: PRESET_COLORS[0] }, { ...defaultOption(1) }];
  });
  const [correctNumber, setCorrectNumber] = useState(initial?.correct_number ?? '');
  const [tolerance, setTolerance] = useState(initial?.tolerance ?? '');
  const [media, setMedia] = useState(initial?.images || []);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');

  const maxOptions = type === 'ordering' || type === 'matching' ? 8 : 6;

  const handleTypeChange = (newType) => {
    setType(newType);
    setError('');
    if (newType === 'numeric') {
      setOptions([]);
    } else if (newType === 'ordering') {
      if (options.length < 2) setOptions([{ text: '', correct_position: 1 }, { text: '', correct_position: 2 }]);
      else setOptions(options.map((o, i) => ({ ...o, correct_position: i + 1 })));
    } else if (newType === 'matching') {
      if (options.length < 2) setOptions([{ text: '', match_text: '' }, { text: '', match_text: '' }]);
      else setOptions(options.map((o) => ({ ...o, match_text: o.match_text || '' })));
    } else if (newType === 'multiple_choice') {
      setOptions((prev) => prev.length >= 2 ? prev : [{ text: '', is_correct: true, color: PRESET_COLORS[0] }, { ...defaultOption(1) }]);
    } else {
      if (options.length < 2) setOptions([{ text: '', is_correct: true, color: PRESET_COLORS[0] }, { ...defaultOption(1) }]);
      else {
        const hasCorrect = options.some(o => o.is_correct);
        if (!hasCorrect) {
          const next = [...options];
          next[0] = { ...next[0], is_correct: true };
          setOptions(next);
        }
      }
    }
  };

  const setOptionField = (i, field, val) => {
    const next = [...options];
    next[i] = { ...next[i], [field]: val };
    setOptions(next);
  };

  const setSingleCorrect = (i) => {
    setOptions(options.map((o, idx) => ({ ...o, is_correct: idx === i })));
  };

  const toggleMultiCorrect = (i) => {
    const next = [...options];
    next[i] = { ...next[i], is_correct: !next[i].is_correct };
    setOptions(next);
  };

  const randomizeColors = () => {
    const shuffled = [...PRESET_COLORS].sort(() => Math.random() - 0.5);
    setOptions(options.map((o, i) => ({ ...o, color: shuffled[i % shuffled.length] })));
  };

  const addOption = () => {
    if (options.length >= maxOptions) return;
    if (type === 'ordering') {
      setOptions([...options, { text: '', correct_position: options.length + 1 }]);
    } else if (type === 'matching') {
      setOptions([...options, { text: '', match_text: '' }]);
    } else {
      setOptions([...options, defaultOption(options.length)]);
    }
  };

  const removeOption = (i) => {
    if (options.length <= 2) return;
    let next = options.filter((_, idx) => idx !== i);
    if (type === 'ordering') {
      next = next.map((o, idx) => ({ ...o, correct_position: idx + 1 }));
    } else if (type === 'single_choice' && !next.some(o => o.is_correct)) {
      next[0].is_correct = true;
    }
    setOptions(next);
  };

  const handleMediaUpload = async (e) => {
    const files = Array.from(e.target.files);
    if (!files.length) return;
    setUploading(true);
    try {
      for (const file of files) {
        const { data } = await uploadImage(file);
        const mediaType = data.type || 'image';
        if (initial?.id) {
          const { data: img } = await addQuestionImage(initial.id, data.url, mediaType);
          setMedia((prev) => [...prev, img]);
        } else {
          setMedia((prev) => [...prev, { url: data.url, type: mediaType, id: Date.now() + Math.random() }]);
        }
      }
    } catch { /* ignore */ }
    setUploading(false);
    e.target.value = '';
  };

  const handleRemoveMedia = async (item) => {
    if (initial?.id && typeof item.id === 'number') {
      await deleteQuestionImage(item.id);
    }
    setMedia((prev) => prev.filter((i) => i !== item));
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    setError('');

    if (!text.trim()) { setError('Введите текст вопроса'); return; }

    if (type === 'numeric') {
      if (correctNumber === '' || correctNumber === null) { setError('Введите правильное число'); return; }
      onSave({
        text: text.trim(), order_num: orderNum, type,
        correct_number: parseFloat(correctNumber),
        tolerance: tolerance !== '' ? parseFloat(tolerance) : 0,
        options: [],
        _media: media.filter((i) => !initial?.id).map((i) => ({ url: i.url, type: i.type || 'image' })),
      });
      return;
    }

    if (options.some((o) => !o.text.trim())) { setError('Заполните все варианты'); return; }

    if (type === 'single_choice') {
      if (!options.some((o) => o.is_correct)) { setError('Отметьте правильный ответ'); return; }
    } else if (type === 'multiple_choice') {
      if (!options.some((o) => o.is_correct)) { setError('Отметьте хотя бы один правильный ответ'); return; }
      if (options.every((o) => o.is_correct)) { setError('Хотя бы один ответ должен быть неправильным'); return; }
    } else if (type === 'matching') {
      if (options.some((o) => !o.match_text?.trim())) { setError('Заполните все пары соотнесения'); return; }
    }

    const mappedOptions = options.map((o, i) => {
      const opt = { text: o.text.trim(), is_correct: !!o.is_correct, color: o.color || '' };
      if (type === 'ordering') opt.correct_position = o.correct_position ?? (i + 1);
      if (type === 'matching') opt.match_text = o.match_text?.trim() || '';
      return opt;
    });

    onSave({
      text: text.trim(), order_num: orderNum, type,
      options: mappedOptions,
      _media: media.filter((i) => !initial?.id).map((i) => ({ url: i.url, type: i.type || 'image' })),
    });
  };

  const isWebOnly = (t) => QUESTION_TYPES.find(qt => qt.value === t)?.webOnly;

  return (
    <form className="question-form" onSubmit={handleSubmit}>
      <h3>{initial ? 'Редактировать вопрос' : 'Новый вопрос'}</h3>

      {error && <div className="error-msg">{error}</div>}

      <div className="form-group">
        <label>Тип вопроса</label>
        <div className="question-type-selector">
          {QUESTION_TYPES.map((qt) => {
            const disabled = quizMode === 'bot' && qt.webOnly;
            return (
              <button
                key={qt.value}
                type="button"
                className={`type-btn${type === qt.value ? ' active' : ''}${disabled ? ' disabled' : ''}`}
                onClick={() => !disabled && handleTypeChange(qt.value)}
                title={disabled ? 'Недоступно в режиме бота' : qt.label}
              >
                {qt.label}
                {qt.webOnly && <span className="web-badge">web</span>}
              </button>
            );
          })}
        </div>
      </div>

      <div className="form-group">
        <label>Текст вопроса</label>
        <input value={text} onChange={(e) => setText(e.target.value)} placeholder="Введите вопрос..." />
      </div>

      <div className="form-group">
        <label>Медиа (фото / аудио / видео)</label>
        <div className="images-row">
          {media.map((item, i) => (
            <div key={i} className="media-preview-wrap">
              {(!item.type || item.type === 'image') && <img src={item.url} alt="" className="img-preview" />}
              {item.type === 'audio' && <audio src={item.url} controls className="audio-preview" />}
              {item.type === 'video' && <video src={item.url} controls className="video-preview" />}
              <button type="button" className="img-remove" onClick={() => handleRemoveMedia(item)}>✕</button>
            </div>
          ))}
          <label className="img-upload-btn">
            {uploading ? '...' : '+ Медиа'}
            <input type="file" accept="image/*,audio/*,video/*" multiple onChange={handleMediaUpload} hidden />
          </label>
        </div>
      </div>

      {type === 'numeric' ? (
        <div className="numeric-editor">
          <div className="form-group">
            <label>Правильное число</label>
            <input type="number" step="any" value={correctNumber} onChange={(e) => setCorrectNumber(e.target.value)} placeholder="Например: 42" />
          </div>
          <div className="form-group">
            <label>Допуск (±)</label>
            <input type="number" step="any" min="0" value={tolerance} onChange={(e) => setTolerance(e.target.value)} placeholder="0 = точное совпадение" />
          </div>
        </div>
      ) : (
        <>
          <div className="options-header-row">
            <label className="options-label">
              {type === 'ordering' ? 'Элементы (в правильном порядке)' :
               type === 'matching' ? 'Пары соотнесения' :
               'Варианты ответа'}
            </label>
            {(type === 'single_choice' || type === 'multiple_choice') && (
              <button type="button" className="btn btn-outline btn-sm" style={{ fontSize: 11 }} onClick={randomizeColors}>🎨 Цвета</button>
            )}
          </div>

          {type === 'matching' ? (
            <div className="matching-editor">
              <div className="matching-header-labels">
                <span>Термин</span><span>Определение</span>
              </div>
              {options.map((opt, i) => (
                <div className="matching-row" key={i}>
                  <input type="text" value={opt.text} onChange={(e) => setOptionField(i, 'text', e.target.value)} placeholder={`Термин ${i + 1}`} />
                  <span className="matching-arrow">↔</span>
                  <input type="text" value={opt.match_text || ''} onChange={(e) => setOptionField(i, 'match_text', e.target.value)} placeholder={`Определение ${i + 1}`} />
                  {options.length > 2 && (
                    <button type="button" className="btn-icon" onClick={() => removeOption(i)}>✕</button>
                  )}
                </div>
              ))}
            </div>
          ) : type === 'ordering' ? (
            <div className="ordering-editor">
              {options.map((opt, i) => (
                <div className="ordering-row" key={i}>
                  <span className="ordering-num">{i + 1}</span>
                  <input type="text" value={opt.text} onChange={(e) => setOptionField(i, 'text', e.target.value)} placeholder={`Элемент ${i + 1}`} />
                  {options.length > 2 && (
                    <button type="button" className="btn-icon" onClick={() => removeOption(i)}>✕</button>
                  )}
                </div>
              ))}
              <p className="ordering-hint">Порядок элементов здесь — правильный. Участники увидят их перемешанными.</p>
            </div>
          ) : (
            <>
              {options.map((opt, i) => (
                <div className="option-row" key={i}>
                  {type === 'single_choice' ? (
                    <input type="radio" name="correct" checked={opt.is_correct} onChange={() => setSingleCorrect(i)} title="Правильный ответ" />
                  ) : (
                    <input type="checkbox" checked={opt.is_correct} onChange={() => toggleMultiCorrect(i)} title="Правильный ответ" />
                  )}
                  <input type="color" value={opt.color || '#e21b3c'} onChange={(e) => setOptionField(i, 'color', e.target.value)} className="color-picker" title="Цвет" />
                  <input type="text" value={opt.text} onChange={(e) => setOptionField(i, 'text', e.target.value)} placeholder={`Вариант ${i + 1}`} style={{ borderLeft: `4px solid ${opt.color || '#ccc'}` }} />
                  {options.length > 2 && (
                    <button type="button" className="btn-icon" onClick={() => removeOption(i)}>✕</button>
                  )}
                </div>
              ))}
            </>
          )}
        </>
      )}

      <div className="question-form-actions">
        <button type="submit" className="btn btn-success btn-sm">Сохранить</button>
        {type !== 'numeric' && options.length < maxOptions && (
          <button type="button" className="btn btn-outline btn-sm" onClick={addOption}>
            + {type === 'matching' ? 'Пара' : type === 'ordering' ? 'Элемент' : 'Вариант'}
          </button>
        )}
        <button type="button" className="btn btn-outline btn-sm" onClick={onCancel}>Отмена</button>
      </div>
    </form>
  );
}
