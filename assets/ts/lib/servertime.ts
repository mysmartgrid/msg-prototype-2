import {now} from './utils';
import {ServerTimeHandler, Socket} from './msg2socket';



const OffsetCount = 25;

export class ServerTime {
    private _offsets : number[];
    private _averageOffset : number;

    constructor(private _socket : Socket) {
        console.log("New ServerTime");
        this._averageOffset = 0;
        this._offsets = new Array<number>(OffsetCount);
        this._offsets.fill(0);

        _socket.onServerTime((servertime) => this._updateOffsets(servertime));
    }


    private _updateOffsets(servertime : number) {
        this._offsets.shift();
        this._offsets.push(now() - servertime);

        this._averageOffset = 0;
        for(var offset of this._offsets) {
            this._averageOffset += offset / OffsetCount;
        }

        console.log("Timeoffset:", this._averageOffset);
    }


    public getOffset() : number {
        return this._averageOffset;
    }

    public now() : number {
        return now() - this._averageOffset;
    }
}


export const ServerTimeFactory =  ["WSUserClient", (socket : Socket) => new ServerTime(socket)];
