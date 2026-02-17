import { useState } from 'react';
import { uploadImage, addQuestionImage, deleteQuestionImage } from '../api/quizzes';

const PRESET_COLORS = ['#e21b3c', '#1368ce', '#d89e00', '#26890c', '#864cbf', '#0aa3b1'];

function randomColor() {
  return PRESET_COLORS[Math.floor(Math.random() * PRESET_COLORS.length)];
}

const defaultOption = (i) => ({ text: '', is_correct: false, color: PRESET_COLORS[i % PRESET_COLORS.length] });

export default function QuestionForm({ initial, orderNum, onSave, onCancel }) {
  const [text, setText] = useState(initial?.text || '');
  const [options, setOptions] = useState(
    initial?.options?.map((o) => ({ text: o.text, is_correct: o.is_correct, color: o.color || PRESET_COLORS[0] })) ||
    [{ text: '', is_correct: true, color: PRESET_COLORS[0] }, { ...defaultOption(1) }]
  );
  const [images, setImages] = useState(initial?.images || []);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');

  const setOptionText = (i, val) => {
    const next = [...options];
    next[i] = { ...next[i], text: val };
    setOptions(next);
  };

  const setCorrect = (i) => {
    setOptions(options.map((o, idx) => ({ ...o, is_correct: idx === i })));
  };

  const setColor = (i, color) => {
    const next = [...options];
    next[i] = { ...next[i], color };
    setOptions(next);
  };

  const randomizeColors = () => {
    const shuffled = [...PRESET_COLORS].sort(() => Math.random() - 0.5);
    setOptions(options.map((o, i) => ({ ...o, color: shuffled[i % shuffled.length] })));
  };

  const addOption = () => {
    if (options.length < 4) setOptions([...options, defaultOption(options.length)]);
  };

  const removeOption = (i) => {
    if (options.length <= 2) return;
    const next = options.filter((_, idx) => idx !== i);
    if (!next.some((o) => o.is_correct)) next[0].is_correct = true;
    setOptions(next);
  };

  const handleImageUpload = async (e) => {
    const files = Array.from(e.target.files);
    if (!files.length) return;
    setUploading(true);
    try {
      for (const file of files) {
        const { data } = await uploadImage(file);
        if (initial?.id) {
          const { data: img } = await addQuestionImage(initial.id, data.url);
          setImages((prev) => [...prev, img]);
        } else {
          setImages((prev) => [...prev, { url: data.url, id: Date.now() + Math.random() }]);
        }
      }
    } catch { /* ignore */ }
    setUploading(false);
    e.target.value = '';
  };

  const handleRemoveImage = async (img) => {
    if (initial?.id && typeof img.id === 'number') {
      await deleteQuestionImage(img.id);
    }
    setImages((prev) => prev.filter((i) => i !== img));
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    setError('');

    if (!text.trim()) { setError('Введите текст вопроса'); return; }
    if (options.some((o) => !o.text.trim())) { setError('Заполните все варианты ответа'); return; }
    if (!options.some((o) => o.is_correct)) { setError('Отметьте правильный ответ'); return; }

    onSave({
      text: text.trim(),
      order_num: orderNum,
      options: options.map((o) => ({ text: o.text.trim(), is_correct: o.is_correct, color: o.color })),
      _images: images.filter((i) => !initial?.id).map((i) => i.url),
    });
  };

  return (
    <form className="question-form" onSubmit={handleSubmit}>
      <h3>{initial ? 'Редактировать вопрос' : 'Новый вопрос'}</h3>

      {error && <div className="error-msg">{error}</div>}

      <div className="form-group">
        <label>Текст вопроса</label>
        <input value={text} onChange={(e) => setText(e.target.value)} placeholder="Введите вопрос..." />
      </div>

      <div className="form-group">
        <label>Изображения</label>
        <div className="images-row">
          {images.map((img, i) => (
            <div key={i} className="img-preview-wrap">
              <img src={img.url} alt="" className="img-preview" />
              <button type="button" className="img-remove" onClick={() => handleRemoveImage(img)}>✕</button>
            </div>
          ))}
          <label className="img-upload-btn">
            {uploading ? '...' : '+ Фото'}
            <input type="file" accept="image/*" multiple onChange={handleImageUpload} hidden />
          </label>
        </div>
      </div>

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <label style={{ fontSize: '13px', fontWeight: 600, color: '#636e72', textTransform: 'uppercase', letterSpacing: '0.5px' }}>
          Варианты ответа
        </label>
        <button type="button" className="btn btn-outline btn-sm" style={{ fontSize: 11 }} onClick={randomizeColors}>🎨 Случайные цвета</button>
      </div>

      {options.map((opt, i) => (
        <div className="option-row" key={i}>
          <input type="radio" name="correct" checked={opt.is_correct} onChange={() => setCorrect(i)} title="Правильный ответ" />
          <input
            type="color"
            value={opt.color || '#e21b3c'}
            onChange={(e) => setColor(i, e.target.value)}
            className="color-picker"
            title="Цвет варианта"
          />
          <input type="text" value={opt.text} onChange={(e) => setOptionText(i, e.target.value)} placeholder={`Вариант ${i + 1}`} style={{ borderLeft: `4px solid ${opt.color || '#ccc'}` }} />
          {options.length > 2 && (
            <button type="button" className="btn-icon" onClick={() => removeOption(i)} title="Удалить">✕</button>
          )}
        </div>
      ))}

      <div className="question-form-actions">
        <button type="submit" className="btn btn-success btn-sm">Сохранить</button>
        {options.length < 4 && (
          <button type="button" className="btn btn-outline btn-sm" onClick={addOption}>+ Вариант</button>
        )}
        <button type="button" className="btn btn-outline btn-sm" onClick={onCancel}>Отмена</button>
      </div>
    </form>
  );
}
