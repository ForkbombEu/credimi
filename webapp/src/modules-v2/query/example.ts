// Define a merger interface
interface MergerStrategy<T extends object = object, U extends object = object> {
	merge(a: T, b: U): unknown;
}

// Implement different strategies
type SimpleMerger<T extends object = object, U extends object = object> = {
	merge(a: T, b: U): T & U;
};

type SmartMerger<T extends object = object, U extends object = object> = {
	merge(a: T, b: U): T & Omit<U, keyof T> & { merged: true };
};

// Your Result type becomes more flexible
type Result<A extends object, B extends object, M extends MergerStrategy<A, B>> = ReturnType<
	M['merge']
>;
