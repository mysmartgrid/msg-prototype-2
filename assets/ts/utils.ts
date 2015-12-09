module Utils {


    export class ExtArray<U> extends Array<U> {

        public contains(element : U) : boolean {
            var i = this.indexOf(element);
            return i !== -1;
        }

        public remove(element : U) : void {
            var i = this.indexOf(element);
            if(i !== -1) {
                this.splice(i,1);
            }
        }

        public removeWhere(pred: (element : U) => boolean) {
            var i = this.findIndex(pred);
            while(i !== -1) {
                this.splice(i,1);
                var i = this.findIndex(pred);
            }
        }



    }
}
