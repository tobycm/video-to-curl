import { Writable, WritableOptions } from "stream";

export default class WritableQueue extends Writable {
  queue: any[];

  constructor(options: WritableOptions) {
    super(options);
    this.queue = [];
  }

  _write(chunk: any, encoding: BufferEncoding, callback: (error?: Error | null) => void) {
    this.queue.push(chunk);
    callback();
  }
}
