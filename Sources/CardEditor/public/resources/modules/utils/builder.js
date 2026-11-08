export default class Builder {
  #func;
  #res;

  constructor(func) {
    this.#func = func;
  }

  get value() {
    this.#res ||= this.#func();
    return this.#res;
  }
}
