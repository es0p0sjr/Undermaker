/* eslint-disable */
const effects = [
    'Magic', 'Future', 'Dodge', 'Dust', 'Taunt', 'Paralyze', 'paralyzed', 'Start of turn', 'End of turn', 'Haste', 'Armor', `Can't attack`, 'Candy', 'Transparency', 'Charge',
    'Fatigue', 'Turbo', 'Ranged', 'Support',
    // Tribes
    'Amalgamates?', 'Bombs?', 'Dogs?', 'Froggits?', 'G followers?', 'Lost souls?', 'Mold', 'Plants?', 'Royal guards?', 'Snails?', 'Spiders?', 'Temmies?', 'Bargains?',
];
const specials = ['ATK', 'DMG', 'HP', 'KR'];
const underlineRegex = new RegExp(`(${effects.join('|')})(?![^{]*})|_([^_]+)_`, 'g');
const colorRegex = new RegExp(`(${specials.join('|')})`, 'g');
const highlightRegex = /\{([^}]+)}/g;
let extras = false;

function showExtras() {
    extras = !extras;
    document.body.classList.toggle('showExtras', extras);
}

function generate(monster = true) {
    gtag('event', `create_${monster ? 'monster':'spell'}`,);
    const container = document.getElementById('cards');
    const wrapper = document.createElement('span');
    wrapper.className = 'cardWrapper';
    // Desc max: 115
    wrapper.innerHTML = `
        <table class="cardBoard ${monster?'monster':'spell'}">
            <tr>
                <td class="name" colspan="3"><span></span><input type="text" maxlength="15" placeholder="Name"></td>
                <td class="cost edit"><span>0</span></td>
            </tr>
            <tr>
                <td class="image" colspan="4">
                    <input type="file" accept="image/*">
                    <img>
                    <img class="tribe none${monster?'':' extra'}" src="./resources/tribes/MONSTER.png">
                </td>
            </tr>
            <tr>
                <td class="description" colspan="4"><div></div><textarea></textarea></td>
            </tr>
            <tr>
                ${monster?'<td class="attack edit"><span>0</span></td>':''}
                <td class="rarity" colspan="${monster?2:4}"><img src="rarity/COMMON.png"></td>
                ${monster?'<td class="health edit"><span>0</span></td>':''}
            </tr>
        </table>
        <span class="footer">undercard.feildmaster.com</span>`;
    const card = wrapper.querySelector('.cardBoard');
    card.oncontextmenu = cardMenu.bind(null, card);
    // Name edit
    const nameCell = wrapper.querySelector('.name');
    nameCell.onclick = editName.bind(nameCell);
    nameCell.querySelector('input').onblur = finalizeName.bind(nameCell);
    // Number edit
    const input = document.createElement('input');
    input.type = 'number';
    input.min = '0';
    wrapper.querySelectorAll('.edit').forEach(function (cell) {
        const clone = input.cloneNode();
        const span = cell.querySelector('span');
        span.after(clone);
        cell.onclick = edit.bind(span, clone);
        clone.onblur = finalizeEdit.bind(clone, span);
    });
    // Description edit
    const description = wrapper.querySelector('.description div');
    const descriptionBox = wrapper.querySelector('.description textarea');
    wrapper.querySelector('.description').onclick = editDescription.bind(description, descriptionBox);
    descriptionBox.onblur = renderDescription.bind(descriptionBox, description);
    // Image edit
    const imageRow = wrapper.querySelector('.image');
    const image = imageRow.querySelector('input');
    imageRow.onclick = () => image.click();
    image.onchange = readImage.bind(image, imageRow.querySelector('img'));
    // Add to document
    container.append(wrapper);
    tippy(nameCell, {
        theme: 'black',
        target: 'input',
        arrow: false,
        content: document.getElementById('selectSoul').innerHTML,
        placement: 'right-start',
        distance: -5,
        onMount(e) {
            e.popper.querySelectorAll('span.selectable').forEach((span) => {
                span.onclick = modifySoul.bind(span, nameCell, e.popper);
            });
        },
        onShow(e) {
            return !monster || extras;
        },
    });
    tippy(imageRow, {
        content: 'Click to Select Image',
        placement: 'top',
        trigger: 'mouseenter',
        size: 'small',
        interactive: false,
    });
    // TODO: Set description keywords to allow insertion
    tippy(descriptionBox);
    tippy(wrapper.querySelector('.rarity'), {
        theme: 'black',
        trigger: 'mouseenter',
        hideOnClick: true,
        arrow: false,
        content: document.getElementById('selectRarity').innerHTML,
        placement: 'top',
        size: 'small',
        distance: -1,
        shouldPopperHideOnBlur() {
            return true;
        },
        onMount(e) {
            const tip = e.popper._tippy;
            e.popper.querySelectorAll('img.selectable').forEach((item) => {
                item.onclick = () => {
                    editEvent('rarity');
                    e.popper.querySelector('img.active').classList.remove('active');
                    item.classList.add('active');
                    tip.reference.querySelector('img').src = item.src;
                    tip.hide();
                };
            });
        },
    });
    const tribe = wrapper.querySelector('.tribe');
    tribe.onclick = (e) => {
        console.log('click');
        e.stopPropagation();
    };
    tippy(tribe, {
        theme: 'black',
        trigger: 'mouseenter focus',
        hideOnClick: false,
        arrow: false,
        content: document.getElementById('selectTribe').innerHTML,
        placement: 'top-end',
        onMount(e) {
            const tip = e.popper._tippy;
            e.popper.querySelectorAll('img.selectable').forEach((item) => {
                item.onclick = () => {
                    editEvent('tribe');
                    e.popper.querySelector('img.active').classList.remove('active');
                    item.classList.add('active');
                    tribe.classList.toggle('none', item.src.includes('MONSTER'));
                    tribe.classList.toggle('smallerIcon', item.src.includes('SPIDER'));
                    tribe.src = item.src;
                    tip.hide();
                };
            });
        },
    });
}

function editName() {
    this.querySelector('span').style.display = 'none';
    this.querySelector('input').focus();
    if (this._tippy) {
        this._tippy.show(0);
    }
}

function finalizeName(e = {}) {
    if (this._tippy) {
        if (e.relatedTarget === this._tippy.popper) return;
        this._tippy.hide(0);
    }
    const span = this.querySelector('span');
    const input = this.querySelector('input');
    if (span.textContent !== input.value) {
        editEvent('name');
        span.textContent = input.value;
    }
    span.style.display = '';
}

function modifySoul(nameCell, popper) {
    const activeSoul = popper.querySelector('span.selectable.active');
    const input = nameCell.querySelector('input');
    if (this === activeSoul) {
        input.focus();
        return;
    } else if (activeSoul) {
        activeSoul.classList.remove('active');
        nameCell.classList.remove(activeSoul.textContent);
    }
    editEvent('soul');
    this.classList.add('active');
    // Modify the cell
    nameCell.classList.add(this.textContent);
    input.focus();
}

function edit(input) {
    this.style.display = 'none';
    input.value = '';
    input.placeholder = this.textContent;
    // Fix firefox's bullshit
    input.keepAlive = true;
    input.focus();
    input.keepAlive = false;
}

function finalizeEdit(span) {
    if (this.keepAlive) return;
    const newValue = this.value || span.textContent;
    if (span.textContent !== newValue) {
        editEvent(span.parentElement.classList[0]);
        span.textContent = newValue;
    }
    span.style.display = '';
}

function editDescription(input) {
    this.style.display = 'none';
    input.focus();
    input._oldValue = input.value;
    input._tippy.show(0);
}

function renderDescription(span, e = {}) {
    const tippy = this._tippy;
    if (e.relatedTarget === tippy.popper) return;
    tippy.hide(0);
    if (this.value !== this._oldValue) {
        editEvent('description');
        const description = this.value
            .replace(underlineRegex, (match, $1, $2) => `<span class="underline">${$2||$1}</span>`)
            .replace(colorRegex, (match, $1) => `<span class="${getClass($1)}">${$1}</span>`)
            .replace(highlightRegex, (match, $1) => `<span class="highlight">${$1}</span>`);
        span.innerHTML = description;
    }
    span.style.display = '';
}

function getClass(keyword) {
    switch (keyword) {
        case 'ATK': return 'attack';
        case 'HP': return 'health';
        case 'DMG': return 'damage';
        case 'KR': return 'poison';
    }
}

function editEvent(type) {
    gtag('event', `edit_${type}`);
}

function readImage(image) {
    if (!(this.files && this.files[0])) return;
    editEvent('image');
    const reader = new FileReader();
    reader.onload = function (e) {
        image.src = e.target.result;
    };
    reader.readAsDataURL(this.files[0]);
}

function saveCard(card) {
    gtag('event', 'save');
    card.parentElement.classList.toggle('saving', true);
    const image = card.querySelector('.image img');
    let removedImage = false;
    if (!image.src) {
        image.parentElement.removeChild(image);
        removedImage = true;
    }
    const description = card.querySelector('.description textarea.tippy-active');
    if (description) { // Render description
        renderDescription.call(description, card.querySelector('.description div'));
    }
    domtoimage.toPng(card.parentElement).then((url) => {
        const link = document.createElement('a');
        link.download = `${card.querySelector('.name input').value||'undercard'}.png`;
        link.href = url;
        link.click();
    }).catch((error) => console.error('Ooops.', error))
    .then(() => {
        card.parentElement.classList.toggle('saving', false);
        if (removedImage) {
            card.querySelector('.image').prepend(image);
        }
    });
}

function cardMenu(card, e) {
    if (e.altKey || e.ctrlKey || e.shiftKey || e.metaKey) return;
    if (e.target === card.querySelector('.description textarea')) return;
    e.stopPropagation();
    const context = document.querySelector('.context');
    context.style.display = 'unset';
    context.style.left = `${e.pageX}px`;
    context.style.top = `${e.pageY}px`;
    // Delete
    context.querySelector('.delete').onclick = () => {
        card.parentElement.remove();
        context.style.display = '';
    };
    // Download
    context.querySelector('.download').onclick = () => {
        saveCard(card);
        context.style.display = '';
    };
    return false;
}

window.onload = () => {
    const context = document.querySelector('.context');
    const close = () => context.style.display = '';
    context.querySelector('.close').onclick = close;
    window.oncontextmenu = close;
    window.onclick = close;
    const div = document.querySelector('#descriptionTip div');
    function addType(type) {
        const el = document.createElement('span');
        el.innerText = type.replace('s?', '');
        div.append(el, ' ');
    }
    effects.forEach(addType);
    specials.forEach(addType);

    tippy.setDefaults({
        content: document.getElementById('descriptionTip').innerHTML,
        placement: 'right-end',
        arrow: true,
        trigger: 'manual',
        hideOnClick: false,
        performance: true,
        shouldPopperHideOnBlur: () => false,
        interactive: true,
    });

    if (/(?:\?|\&)extras\b/i.test(location.search)) {
        showExtras();
    }

    document.querySelector('#buttons').style.display = 'block';
    document.querySelector('#loading').remove();
    new SimpleToast('Update your browser to use the improved editor!');
};

(() => {
    const script = document.createElement('script');
    script.src = 'https://unpkg.com/tippy.js@3.4.1/dist/tippy.all.js';
    document.head.append(script);
})();
