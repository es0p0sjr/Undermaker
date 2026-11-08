import style from './card.css' assert { type: 'css' };
import BaseElement from './base.js';
import { Elements } from './types.js';

document.adoptedStyleSheets.push(style);

export default class Card extends BaseElement {
  /** @type {number | undefined} */
  #attack;
  cost = 0;
  /** @type {string[]} */
  effects = [];
  /** @type {number | undefined} */
  #health;
  image = '';
  rarity = '';
  soul = '';
  /** @type {string[]} */
  tribes = [];

  constructor({
    attack,
    health,
    // soul = '',
    ...rest
  } = {}) {
    super({
      ...rest,
      type: Elements.Card,
    });
    this.#attack = attack;
    this.#health = health;
    // this.#soul = soul;
  }

  get attack() {
    return this.#attack;
  }

  set attack(value = 0) {
    if (this.isSpell) throw new Error('Adding attack to spell');
    if (value < 0) throw new Error('Invalid attack value');
    this.#attack = value;
  }

  get health() {
    return this.#health;
  }

  set health(value = 0) {
    if (this.isSpell) throw new Error('Adding health to spell');
    if (value < 0) throw new Error('Invalid health value');
    this.#health = value;
  }

  setMonster() {
    if (this.#health !== undefined) return;
    this.#health = 0;
    this.#attack = 0;
  }

  get isSpell() {
    return this.#health === undefined;
  }

  toJSON() {
    const {
      attack,
      health,
    } = this;
    return {
      ...super.toJSON(),
      attack,
      health,
    };
  }
}